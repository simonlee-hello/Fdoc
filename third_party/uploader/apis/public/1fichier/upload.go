package fichier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"uploader/apis"
	"uploader/apis/methods"
	"uploader/utils"
)

const (
	ftpServer   = "ftp.1fichier.com:21"
	ftpUser     = "1fichierisgood"
	ftpPassword = "1fichier"
)

var domains = []string{
	"1fichier.com",
	"alterupload.com",
	"cjoint.net",
	"desfichiers.com",
	"dfichiers.com",
	"megadl.fr",
	"mesfichiers.org",
	"piecejointe.net",
	"pjointe.com",
	"tenvoi.com",
	"dl4free.com",
}

var (
	extractDownload = regexp.MustCompile(buildDomainRegex(domains, `\?\w+`))
	extractRemove   = regexp.MustCompile(buildDomainRegex(domains, `remove/[\w/]+`))
)

func (b *fichier) DoUpload(name string, size int64, file io.Reader) error {
	var lastErr error
	if b.useFTP {
		lastErr = b.uploadViaFTP(name, file)
		if lastErr != nil {
			fmt.Fprintf(os.Stderr, "ftp: %v\n", lastErr)
		} else {
			b.resp = "ftp ok"
			b.remove = ""
			return nil
		}
	}
	return b.uploadViaHTTP(name, size, file)
}

func (b *fichier) PostUpload(string, int64) (string, error) {
	if b.resp != "" {
		fmt.Println(b.resp)
	}
	if b.pwd != "" && !apis.QuietMode {
		fmt.Fprintf(os.Stderr, "password: %s\n", b.pwd)
	}
	if b.remove != "" && !apis.QuietMode {
		fmt.Fprintf(os.Stderr, "remove: %s\n", b.remove)
	}
	return b.resp, nil
}

func (b fichier) newMultipartUpload(config uploadConfig) ([]byte, error) {
	if config.debug {
		log.Printf("start upload")
	}
	client := methods.NewClient(0)

	byteBuf := &bytes.Buffer{}
	writer := multipart.NewWriter(byteBuf)
	_ = writer.WriteField("mail", config.email)
	//_ = writer.WriteField("mails", "")
	//_ = writer.WriteField("message", "")
	_ = writer.WriteField("domain", config.domainID)
	_ = writer.WriteField("dpass", config.password)

	_, err := writer.CreateFormFile("file[]", config.fileName)
	if err != nil {
		return nil, err
	}

	writerLength := byteBuf.Len()
	writerBody := make([]byte, writerLength)
	_, _ = byteBuf.Read(writerBody)
	_ = writer.Close()

	lastBoundary := fmt.Sprintf("\r\n--%s--\r\n", writer.Boundary())
	totalSize := int64(writerLength) + config.fileSize + int64(len(lastBoundary))
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
		_, _ = fmt.Fprintf(partW, lastBoundary)
		_ = partW.Close()
	}()

	req, err := http.NewRequest("POST", config.uploadURL, partR)
	if err != nil {
		return nil, err
	}
	req.ContentLength = totalSize
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	req.Header.Set("Referer", "https://1fichier.com/")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", b.apiKey))

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
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		if config.debug {
			log.Printf("read response returns: %v", body)
		}
		return nil, err
	}
	_ = resp.Body.Close()
	// str, _, _ := transform.String(japanese.EUCJP.NewDecoder(), string(body))
	if config.debug {
		log.Printf("returns: %v", string(body))
	}
	return body, nil
}

func (b *fichier) uploadViaHTTP(name string, size int64, file io.Reader) error {
	fmt.Println("upload Via HTTP...")
	var lastErr error
	for domainId, _ := range domains {
		uploadURL, err := getUploadURL(domains[0])
		if err != nil {
			return err
		}
		body, err := b.newMultipartUpload(uploadConfig{
			fileSize:   size,
			fileName:   name,
			fileReader: file,
			debug:      apis.DebugMode,
			email:      b.email,
			password:   b.pwd,
			uploadURL:  strings.TrimSpace(string(uploadURL)),
			apiKey:     b.apiKey,
			domainID:   strconv.Itoa(domainId),
		})

		if err != nil {
			lastErr = fmt.Errorf("upload returns error: %s", err)
			continue
		}

		b.resp = extractDownload.FindString(string(body))
		b.remove = extractRemove.FindString(string(body))
		return nil
	}
	return lastErr
}

func (b *fichier) uploadViaFTP(fileName string, file io.Reader) error {
	return utils.FTPUpload(ftpServer, ftpUser, ftpPassword, fileName, file)
}

func getUploadURL(domain string) (string, error) {
	url := fmt.Sprintf("https://api.%s/v1/upload/get_upload_server.cgi", domain)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", apis.DefaultUA)

	// 发送请求
	client := methods.NewClient(0)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// 解析JSON响应
	var uploadServerResp uploadServerResponse
	err = json.Unmarshal(body, &uploadServerResp)
	if err != nil {
		return "", err
	}

	// 构建上传URL
	uploadURL := fmt.Sprintf("https://%s/upload.cgi?id=%s", uploadServerResp.URL, uploadServerResp.ID)
	return uploadURL, nil
}

func buildDomainRegex(domains []string, pathPattern string) string {
	escapedDomains := make([]string, len(domains))
	for i, domain := range domains {
		escapedDomains[i] = regexp.QuoteMeta(domain)
	}
	domainPattern := strings.Join(escapedDomains, "|")
	return fmt.Sprintf(`https://(?:%s)/%s`, domainPattern, pathPattern)
}
