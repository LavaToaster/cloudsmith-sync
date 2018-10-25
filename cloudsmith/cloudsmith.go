package cloudsmith

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cloudsmith-io/cloudsmith-api/bindings/go/src"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

type Error struct {
	Detail string `json:"detail"`
}

type Client struct {
	Files         cloudsmith_api.FilesApi
	Packages      cloudsmith_api.PackagesApi
	KnownVersions []string
}

func NewClient(apiKey string) *Client {
	configuration := cloudsmith_api.NewConfiguration()
	configuration.AddDefaultHeader("X-Api-Key", apiKey)

	return &Client{
		Files: cloudsmith_api.FilesApi{
			Configuration: configuration,
		},
		Packages: cloudsmith_api.PackagesApi{
			Configuration: configuration,
		},
	}
}

func (c *Client) UploadComposerPackage(owner, repo, artifactPath string) (csPkg *cloudsmith_api.ModelPackage, error error) {
	fileName := filepath.Base(artifactPath)

	// Get upload details from Cloudsmith (which is a pre-signed s3 upload)
	upload, rawUpload, err := c.Files.FilesCreate(owner, repo, cloudsmith_api.FilesCreate{
		Filename:    fileName,
		Md5Checksum: calculateMd5Checksum(artifactPath),
	})

	if err := checkForCloudsmithRequestError(rawUpload, err); err != nil {
		return csPkg, err
	}

	// Convert the upload interface{} to map[string]string
	params := getParams(upload.UploadFields)

	// Prepare request to upload to S3 based on data given from Cloudsmith
	req, err := newS3UploadRequest(upload.UploadUrl, params, "file", artifactPath)

	if err != nil {
		return csPkg, err
	}

	// Perform the upload
	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return csPkg, err
	}

	if resp.StatusCode >= 300 {
		return csPkg, errors.New("s3 file upload failed")
	}

	// Alright, the file uploaded, now to create a package on Cloudsmith and
	// link it to the file
	pkg, rawPkg, err := c.Packages.PackagesUploadComposer(owner, repo, cloudsmith_api.PackagesUploadComposer{
		PackageFile: upload.Identifier,
	})

	if err := checkForCloudsmithRequestError(rawPkg, err); err != nil {
		return csPkg, err
	}

	return pkg, nil
}

func (c *Client) LoadPackages(owner, repo string) error {
	pageSize := 100
	page := 1

	for {
		pkgs, rawList, err := c.Packages.PackagesList(owner, repo, int32(page), int32(pageSize), "status:completed format:composer")

		if err := checkForCloudsmithRequestError(rawList, err); err != nil {
			// If the error is because of a 404, we've reached the end of the list!
			if rawList.StatusCode == 404 {
				break
			}

			return err
		}

		for _, pkg := range pkgs {
			c.KnownVersions = append(c.KnownVersions, pkg.Name+":"+pkg.Version)
		}

		if len(pkgs) < pageSize {
			break
		}

		page++
	}

	return nil
}

func (c *Client) RemoteCheckPackageExists(owner, repo, name, version string) (bool, error) {
	searchTerm := fmt.Sprintf("name:%s version:%s format:composer", name, version)

	pkgs, rawList, err := c.Packages.PackagesList(owner, repo, 1, 1, searchTerm)

	if err := checkForCloudsmithRequestError(rawList, err); err != nil {
		// If the error is because of a 404, we've reached the end of the list! or there is nothing to deal with
		if rawList.StatusCode == 404 {
			return false, nil
		}

		return false, err
	}

	return len(pkgs) != 0, nil
}

func (c *Client) DeletePackageIfExists(owner, repo, name, version string) error {
	searchTerm := fmt.Sprintf("name:%s version:%s status:completed format:composer", name, version)

	pkgs, rawList, err := c.Packages.PackagesList(owner, repo, 1, 1, searchTerm)

	if err := checkForCloudsmithRequestError(rawList, err); err != nil {
		// If the error is because of a 404, we've reached the end of the list! or there is nothing to deal with
		if rawList.StatusCode == 404 {
			return nil
		}

		return err
	}

	if len(pkgs) == 0 {
		return nil
	}

	// Delete the first matching version
	pkg := pkgs[0]

	c.Packages.PackagesDelete(owner, repo, strconv.Itoa(int(pkg.Identifier)))

	return nil
}

func (c *Client) RetryFailed(owner, repo string) error {
	pkgs, rawList, err := c.Packages.PackagesList(owner, repo, 1, 100, "status:failed format:composer")

	if err := checkForCloudsmithRequestError(rawList, err); err != nil {
		// If the error is because of a 404, we've reached the end of the list! or there is nothing to deal with
		if rawList.StatusCode != 404 {
			return nil
		}
	}

	if len(pkgs) == 0 {
		return nil
	}

	for _, pkg := range pkgs {
		c.Packages.PackagesResync(owner, repo, strconv.Itoa(int(pkg.Identifier)))
	}

	return nil
}

func (c *Client) IsAwareOfPackage(name string, version string) bool {
	for _, knownVersion := range c.KnownVersions {
		if knownVersion == name+":"+version {
			return true
		}
	}

	return false
}

func checkForCloudsmithRequestError(response *cloudsmith_api.APIResponse, err error) error {
	// just straight up return err if it isn't nil
	if err != nil {
		return err
	}

	// Check for 4xx 5xx responses as those *should* hopefully be in the error
	// format described in their documentation :)
	if response.StatusCode >= 400 {
		var cmError Error

		json.Unmarshal(response.Payload, cmError)

		return errors.New(cmError.Detail)
	}

	if response.StatusCode >= 300 && response.StatusCode < 400 {
		return errors.New("api request got a redirect back for some reason")
	}

	if response.StatusCode < 200 {
		return errors.New("request got a status code the 100 range")
	}

	// All fine :D
	return nil
}

func calculateMd5Checksum(filePath string) string {
	f, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}

	return hex.EncodeToString(h.Sum(nil))
}

func getParams(fields interface{}) map[string]string {
	params := make(map[string]string)
	for key, value := range fields.(map[string]interface{}) {
		strKey := fmt.Sprintf("%v", key)
		strValue := fmt.Sprintf("%v", value)

		params[strKey] = strValue
	}
	return params
}

func newS3UploadRequest(uri string, params map[string]string, paramName, path string) (*http.Request, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}

	part, err := writer.CreateFormFile(paramName, filepath.Base(path))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", uri, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, err
}
