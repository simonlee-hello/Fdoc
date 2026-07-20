package gofile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"

	"uploader/apis"
	"uploader/apis/methods"
)

const (
	getServer = "https://api.gofile.io/servers"
)

func (b *goFile) InitUpload(_ []string, sizes []int64) error {

	err := b.selectServer()
	if err != nil {
		return err
	}

	return nil
}

func smallParser(body *http.Response, result any) error {
	data, err := io.ReadAll(body.Body)
	if err != nil {
		return fmt.Errorf("read body returns error: %v", err)
	}

	if apis.DebugMode {
		log.Printf("parsing json: %s", string(data))
	}

	if err := json.Unmarshal(data, result); err != nil {
		return fmt.Errorf("parse body returns error: %v", err)
	}

	if apis.DebugMode {
		log.Printf("parsed data: %+v", result)
	}
	return nil
}

func (b *goFile) selectServer() error {

	if !apis.QuietMode {
		fmt.Fprint(os.Stderr, "selecting server...")
	}
	body, err := methods.Get(getServer)
	if err != nil {
		return fmt.Errorf("request %s: %v", getServer, err)
	}

	data, err := io.ReadAll(body.Body)
	if err != nil {
		return err
	}
	_ = body.Body.Close()

	var sevData respBody
	if err := json.Unmarshal(data, &sevData); err != nil {
		return err
	}
	if !apis.QuietMode {
		fmt.Fprintf(os.Stderr, " %s\n", strings.TrimSpace(sevData.Data.Servers[0].Name))
	}
	b.serverLink = fmt.Sprintf("https://%s.gofile.io/contents/uploadfile", strings.TrimSpace(sevData.Data.Servers[0].Name))
	return nil
}

func (b *goFile) DoUpload(name string, size int64, file io.Reader) error {
	body, err := b.newMultipartUpload(uploadConfig{
		fileSize:   size,
		fileName:   name,
		fileReader: file,
		debug:      apis.DebugMode,
	})
	if err != nil {
		return fmt.Errorf("upload returns error: %s", err)
	}

	var respData uploadResp
	if err := json.Unmarshal(body, &respData); err != nil {
		if apis.DebugMode {
			log.Printf("parse response error: %v", err)
		}
		return err
	}
	if respData.Status != "" && respData.Status != "ok" {
		return fmt.Errorf("upload failed: %s", string(body))
	}

	b.downloadLink = respData.Data.DownloadPage
	if b.Config.SingleMode {
		if b.userToken == "" && respData.Data.GuestToken != "" {
			b.userToken = respData.Data.GuestToken
		}
		if b.folderID == "" && respData.Data.ParentFolder != "" {
			b.folderID = respData.Data.ParentFolder
		}
	}
	return nil
}

func (b *goFile) PostUpload(string, int64) (string, error) {
	if b.Config.SingleMode {
		return "", nil
	}
	fmt.Println(b.downloadLink)
	return b.downloadLink, nil
}

func (b *goFile) FinishUpload([]string) (string, error) {
	if b.Config.SingleMode && b.downloadLink != "" {
		fmt.Println(b.downloadLink)
		return b.downloadLink, nil
	}
	return "", nil
}
func (b *goFile) newMultipartUpload(config uploadConfig) ([]byte, error) {
	if config.debug {
		log.Printf("start upload")
	}
	client := methods.NewClient(0)

	byteBuf := &bytes.Buffer{}
	writer := multipart.NewWriter(byteBuf)
	_ = writer.WriteField("token", b.userToken)
	_ = writer.WriteField("folderId", b.folderID)
	_, err := writer.CreateFormFile("file", config.fileName)
	if err != nil {
		return nil, err
	}

	writerLength := byteBuf.Len()
	writerBody := make([]byte, writerLength)
	_, _ = byteBuf.Read(writerBody)
	_ = writer.Close()

	boundary := byteBuf.Len()
	lastBoundary := make([]byte, boundary)
	_, _ = byteBuf.Read(lastBoundary)

	totalSize := int64(writerLength) + config.fileSize + int64(boundary)
	partR, partW := io.Pipe()

	go func() {
		_, _ = partW.Write(writerBody)
		for {
			buf := make([]byte, 256)
			nr, err := io.ReadFull(config.fileReader, buf)
			if nr <= 0 {
				break
			}
			if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
				fmt.Println(err)
				break
			}
			if nr > 0 {
				_, _ = partW.Write(buf[:nr])
			}
		}
		_, _ = partW.Write(lastBoundary)
		_ = partW.Close()
	}()

	req, err := http.NewRequest("POST", b.serverLink, partR)
	if err != nil {
		return nil, err
	}
	req.ContentLength = totalSize
	req.Header.Set("content-length", strconv.FormatInt(totalSize, 10))
	req.Header.Set("content-type", fmt.Sprintf("multipart/form-data; boundary=%s", writer.Boundary()))
	req.Header.Set("User-Agent", apis.DefaultUA)
	if config.debug {
		log.Printf("header: %v", req.Header)
	}
	resp, err := client.Do(req)
	if err != nil {
		if config.debug {
			log.Printf("do requests returns error: %v", err)
		}
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if config.debug {
			log.Printf("read response returns: %v", err)
		}
		return nil, err
	}

	_ = resp.Body.Close()
	if config.debug {
		log.Printf("returns: %v", string(body))
	}

	return body, nil
}
