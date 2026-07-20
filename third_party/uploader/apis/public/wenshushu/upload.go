package wenshushu

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"uploader/apis"
	"uploader/apis/methods"
	"uploader/crypto"
	"uploader/utils"
)

const (
	anonymous = "https://www.wenshushu.cn/ap/login/anonymous"
	addSend   = "https://www.wenshushu.cn/ap/task/addsend"
	getUpID   = "https://www.wenshushu.cn/ap/uploadv2/getupid"
	getUpURL  = "https://www.wenshushu.cn/ap/uploadv2/psurl"
	complete  = "https://www.wenshushu.cn/ap/uploadv2/complete"
	process   = "https://www.wenshushu.cn/ap/ufile/getprocess"
	finish    = "https://www.wenshushu.cn/ap/task/copysend"
	timeToken = "https://www.wenshushu.cn/ag/time"
	userInfo  = "https://www.wenshushu.cn/ap/user/userinfo"
	userStor  = "https://www.wenshushu.cn/ap/user/storage"
)

func (b *wssTransfer) InitUpload(_ []string, sizes []int64) error {
	if b.Config.SingleMode {
		totalSize := int64(0)
		for _, v := range sizes {
			totalSize += v
		}
		return b.initUpload(totalSize, len(sizes))
	}
	return nil
}

func (b *wssTransfer) initUpload(totalSize int64, totalCount int) error {

	config, err := b.getSendConfig(totalSize, totalCount)
	if err != nil {
		return err
	}
	b.baseConf = *config
	return nil
}

func (b *wssTransfer) PreUpload(_ string, size int64) error {
	if !b.Config.SingleMode {
		return b.initUpload(size, 1)
	}
	return nil
}

func (b wssTransfer) DoUpload(name string, size int64, file io.Reader) error {
	// Library callers (and probe) may leave Config zeroed; match CLI defaults.
	if b.Config.BlockSize <= 0 {
		b.Config.BlockSize = 1048576
	}
	if b.Config.Parallel <= 0 {
		b.Config.Parallel = 2
	}
	if b.Config.Interval <= 0 {
		b.Config.Interval = 10
	}

	if size/int64(b.Config.BlockSize) > 10000 {
		b.Config.BlockSize = int(size / 10000)
		if b.Config.BlockSize <= 0 {
			b.Config.BlockSize = 1
		}
		fmt.Printf("blocksize too small, set to %d\n", b.Config.BlockSize)
	}

	wg := new(sync.WaitGroup)
	ch := make(chan *uploadPart)
	for i := 0; i < b.Config.Parallel; i++ {
		go b.uploader(&ch, b.baseConf)
	}
	part := int64(0)
	for {
		part++
		buf := make([]byte, b.Config.BlockSize)
		nr, err := io.ReadFull(file, buf)
		if nr <= 0 {
			break
		}
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			fmt.Println(err)
			break
		}
		if nr > 0 {
			wg.Add(1)
			ch <- &uploadPart{
				content: buf[:nr],
				count:   part,
				name:    name,
				wg:      wg,
			}
		}
	}

	wg.Wait()
	close(ch)
	// finish upload
	err := b.finishUpload(b.baseConf, name)
	if err != nil {
		return fmt.Errorf("finishUpload returns error: %v", err)
	}

	return nil
}

func (b wssTransfer) PostUpload(string, int64) (string, error) {
	if !b.Config.SingleMode {
		return b.completeUpload(b.baseConf)
	}
	return "", nil
}

func (b wssTransfer) FinishUpload([]string) (string, error) {
	if b.Config.SingleMode {
		return b.completeUpload(b.baseConf)
	}
	return "", nil
}

func (b wssTransfer) uploader(ch *chan *uploadPart, config sendConfigBlock) {
	for item := range *ch {
		d, _ := json.Marshal(map[string]any{
			"ispart": true,
			"fname":  item.name,
			"partnu": item.count,
			"fsize":  b.Config.BlockSize,
			"upId":   config.UploadID,
		})
		uploadTicket, err := newRequest(getUpURL, string(d), requestConfig{
			debug:    apis.DebugMode,
			retry:    0,
			timeout:  time.Duration(b.Config.Interval) * time.Second,
			modifier: addToken(config.Token),
		})
		if err != nil {
			if apis.DebugMode {
				log.Printf("get upload url request returns error: %v", err)
			}
			*ch <- item
			continue
		}

		client := methods.NewClient(time.Duration(b.Config.Interval) * time.Second)
		data := new(bytes.Buffer)
		data.Write(item.content)
		if apis.DebugMode {
			log.Printf("part %d start uploading", item.count)
			log.Printf("part %d posting %s", item.count, uploadTicket.Data.URL)
		}
		req, err := http.NewRequest("PUT", uploadTicket.Data.URL, data)
		if err != nil {
			if apis.DebugMode {
				log.Printf("build request returns error: %v", err)
			}
			*ch <- item
			continue
		}
		req.Header.Set("content-type", "application/octet-stream")
		resp, err := client.Do(req)
		if err != nil {
			if apis.DebugMode {
				log.Printf("failed uploading part %d error: %v (retrying)", item.count, err)
			}
			*ch <- item
			continue
		}
		_, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			if apis.DebugMode {
				log.Printf("failed uploading part %d error: %v (retrying)", item.count, err)
			}
			*ch <- item
			continue
		}

		_ = resp.Body.Close()

		if apis.DebugMode {
			log.Printf("part %d finished.", item.count)
		}
		item.wg.Done()
	}

}

func (b wssTransfer) finishUpload(config sendConfigBlock, name string) error {
	if apis.DebugMode {
		log.Println("finish upload...")
		log.Println("step1 -> complete")
	}
	d, _ := json.Marshal(map[string]any{
		"ispart": true,
		"fname":  name,
		"location": map[string]string{
			"boxid": config.Bid,
			"preid": config.UFileID,
		},
		"upId": config.UploadID,
	})

	body, err := newRequest(complete, string(d), requestConfig{
		debug:    apis.DebugMode,
		retry:    0,
		timeout:  time.Duration(b.Config.Interval) * time.Second,
		modifier: addToken(config.Token),
	})
	if err != nil {
		return err
	}
	if body.Message != "success" {
		return fmt.Errorf("upload failed returns: %s", body.Message)
	}
	return nil
}

func (b wssTransfer) completeUpload(config sendConfigBlock) (string, error) {
	if apis.DebugMode {
		log.Println("complete upload...")
		log.Println("step1 -> process")
	}
	d, _ := json.Marshal(map[string]string{"processId": config.UploadID})
	for {
		body, err := newRequest(process, string(d), requestConfig{
			debug:    apis.DebugMode,
			retry:    0,
			timeout:  time.Duration(b.Config.Interval) * time.Second,
			modifier: addToken(config.Token),
		})
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		if body.Data.R == "success" {
			break
		}
		time.Sleep(time.Second)
	}

	if apis.DebugMode {
		log.Println("step2 -> finish(copySend)")
	}
	d, _ = json.Marshal(map[string]string{
		"bid":     config.Bid,
		"ufileid": config.UFileID,
		"tid":     config.Tid,
	})
	body, err := newRequest(finish, string(d), requestConfig{
		debug:    apis.DebugMode,
		retry:    0,
		timeout:  time.Duration(b.Config.Interval) * time.Second,
		modifier: addToken(config.Token),
	})
	if err != nil {
		return "", err
	}
	if body.Message != "success" {
		return "", fmt.Errorf("status != success")
	}
	if body.Data.PublicURL == "" {
		return "", fmt.Errorf("empty public url")
	}
	if !apis.QuietMode && body.Data.ManageURL != "" {
		fmt.Fprintf(os.Stderr, "manage: %s\n", body.Data.ManageURL)
	}
	fmt.Println(body.Data.PublicURL)
	return body.Data.PublicURL, nil
}

func (b wssTransfer) getTicket() (string, error) {
	if b.Config.Token != "" {
		return b.Config.Token, nil
	}
	if apis.DebugMode {
		log.Println("getToken...")
	}
	config, err := newRequest(anonymous, "{\"dev_info\":\"{}\"}", requestConfig{
		debug:    apis.DebugMode,
		retry:    0,
		timeout:  time.Duration(b.Config.Interval) * time.Second,
		modifier: addToken(""),
	})
	if err != nil {
		return "", err
	}
	t := config.Data.Token

	return t, nil
}

func (b wssTransfer) encrypt(ts, token string, data []byte) (string, error) {
	md5Hash := md5.New()
	md5Hash.Write(data)
	md5Hash.Write([]byte(token))
	md5Str := hex.EncodeToString(md5Hash.Sum(nil))
	hashStr := []byte(utils.Base58Encode([]byte(md5Str)))
	var timeIV []byte
	for _, k := range utils.Reverse(ts)[:5] {
		pos, _ := strconv.Atoi(string(k))
		timeIV = append(timeIV, ts[pos])
	}
	timeIV = append(timeIV, []byte("000")...)
	enc, err := crypto.EncryptDESCBC(hashStr, timeIV, timeIV)
	if err != nil {
		return "", err
	}
	b64Enc := base64.StdEncoding.EncodeToString(enc)
	return b64Enc, nil
}

func (b wssTransfer) getSendConfig(totalSize int64, totalCount int) (*sendConfigBlock, error) {
	ticket, err := b.getTicket()
	if err != nil {
		return nil, err
	}

	if apis.DebugMode {
		log.Println("step 1/3 timeToken")
	}
	req, err := http.NewRequest("GET", timeToken, nil)
	if err != nil {
		return nil, err
	}
	addHeaders(req)
	resp, err := methods.NewClient(time.Duration(b.Config.Interval) * time.Second).Do(req)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		_ = resp.Body.Close()
		return nil, err
	}
	_ = resp.Body.Close()
	if apis.DebugMode {
		log.Printf("returns: %v", string(body))
	}
	respDat := new(timeConfigResp)
	err = json.Unmarshal(body, respDat)
	if err != nil || respDat.Message != "success" {
		return nil, fmt.Errorf("failed get timeToken, %v", err)
	}
	if apis.DebugMode {
		log.Printf("%+v", respDat)
	}

	if apis.DebugMode {
		log.Println("step 2/3 addSend")
	}
	// encoding/json sorts map keys; a-code is MD5 over exact body bytes.
	data, _ := json.Marshal(map[string]any{
		"downPreCountLimit":  0,
		"expire":             "1",
		"fileDisplay":        0,
		"file_count":         totalCount,
		"file_size":          totalSize,
		"isextension":        false,
		"notDownload":        false,
		"notPreview":         false,
		"notSaveTo":          false,
		"pwd":                b.Config.PassCode,
		"recvs":              []string{"social", "public"},
		"remark":             "",
		"sender":             "",
		"task_traffic_limit": "",
		"trafficStatus":      0,
	})

	encData, err := b.encrypt(respDat.Data.Time, ticket, data)
	if err != nil {
		return nil, err
	}

	config, err := newRequest(addSend, string(data), requestConfig{
		debug:    apis.DebugMode,
		retry:    0,
		timeout:  time.Duration(b.Config.Interval) * time.Second,
		modifier: addToken(ticket, respDat.Data.Time, encData),
	})
	if err != nil {
		return nil, err
	}

	if apis.DebugMode {
		log.Println("step 3/3 getUpID")
	}
	data, _ = json.Marshal(map[string]any{
		"boxid":      config.Data.Bid,
		"preid":      config.Data.UFileID,
		"linkid":     config.Data.Tid,
		"utype":      "sendcopy",
		"originUpid": "",
		"length":     totalSize,
		"count":      totalCount,
	})
	upData, err := newRequest(getUpID, string(data), requestConfig{
		debug:    apis.DebugMode,
		retry:    0,
		timeout:  time.Duration(b.Config.Interval) * time.Second,
		modifier: addToken(ticket),
	})
	if err != nil {
		return nil, err
	}
	config.Data.UploadID = upData.Data.UploadID
	config.Data.Token = ticket
	if apis.DebugMode {
		log.Printf("%+v", config.Data)
	}
	return &config.Data, nil
}

func newRequest(link string, postBody string, config requestConfig) (*sendConfigResp, error) {
	if config.debug {
		log.Printf("endpoint: %s", link)
		log.Printf("postBody: %s", postBody)

	}

	client := methods.NewClient(config.timeout)
	req, err := http.NewRequest("POST", link, bytes.NewReader([]byte(postBody)))
	if err != nil {
		if config.debug {
			log.Printf("build request returns error: %v", err)
		}
		if config.retry > 3 {
			return nil, fmt.Errorf("request %s error: on http.NewRequest, retry exhausted.\n  Last err is %s", link, err.Error())
		}
		config.retry++
		return newRequest(link, postBody, config)
	}
	config.modifier(req)
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if config.debug {
		log.Printf("%+v", req.Header)
	}
	resp, err := client.Do(req)
	if err != nil {
		if config.debug {
			log.Printf("do request returns error: %v", err)
		}
		if config.retry > 3 {
			return nil, fmt.Errorf("request %s error: on client.Do, retry exhausted.\n  Last err is %s", link, err.Error())
		}
		config.retry++
		return newRequest(link, postBody, config)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		if config.debug {
			log.Printf("read response returns: %v", err)
		}
		if config.retry > 3 {
			return nil, fmt.Errorf("request %s error: on resp.ReadAll, retry exhausted.\n  Last err is %s", link, err.Error())
		}
		config.retry++
		return newRequest(link, postBody, config)
	}
	_ = resp.Body.Close()
	if config.debug {
		log.Printf("returns: %v", string(body))
	}

	respDat := new(sendConfigResp)
	err = json.Unmarshal(body, respDat)
	if err != nil || respDat.Message != "success" || respDat.Code != 0 {
		if config.retry > 3 {
			return nil, fmt.Errorf("request %s error: on resp.Parse, retry exhausted.\n  Last Resp is %s", link, body)
		}
		config.retry++
		return newRequest(link, postBody, config)
	}
	if config.debug {
		log.Printf("%+v", respDat)
	}
	return respDat, nil
}

func addToken(added ...string) func(req *http.Request) {
	return func(req *http.Request) {
		addHeaders(req)
		// Preserve exact header names; edge treats A-code case-sensitively.
		req.Header["X-TOKEN"] = []string{added[0]}
		if len(added) >= 2 {
			req.Header["Req-Time"] = []string{added[1]}
		}
		if len(added) >= 3 {
			req.Header["A-code"] = []string{added[2]}
		}
		req.Header.Set("Content-Type", "application/json")
	}
}

func addHeaders(req *http.Request) {
	// Keep headers minimal — `authority` triggers user-agent error:-3.
	req.Header.Set("Prod", "com.wenshushu.web.pc")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", "https://www.wenshushu.cn/")
	req.Header.Set("Accept-Language", "zh-CN, zh-Hans-CN;q=0.9")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Origin", "https://www.wenshushu.cn")
}
