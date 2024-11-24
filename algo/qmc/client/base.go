package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type QQMusic struct {
	http *http.Client
}

func (c *QQMusic) rpcDoRequest(ctx context.Context, reqBody any) ([]byte, error) {
	reqBodyBuf, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("qqMusicClient[rpcDoRequest] marshal request: %w", err)
	}

	const endpointURL = "https://u.y.qq.com/cgi-bin/musicu.fcg"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		endpointURL+fmt.Sprintf("?pcachetime=%d", time.Now().Unix()),
		bytes.NewReader(reqBodyBuf),
	)
	if err != nil {
		return nil, fmt.Errorf("qqMusicClient[rpcDoRequest] create request: %w", err)
	}

	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "zh-CN")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MSIE 9.0; Windows NT 6.1; WOW64; Trident/5.0)")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// req.Header.Set("Accept-Encoding", "gzip, deflate")

	reqp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("qqMusicClient[rpcDoRequest] send request: %w", err)
	}
	defer reqp.Body.Close()

	respBodyBuf, err := io.ReadAll(reqp.Body)
	if err != nil {
		return nil, fmt.Errorf("qqMusicClient[rpcDoRequest] read response: %w", err)
	}

	return respBodyBuf, nil
}

type rpcRequest struct {
	Method string `json:"method"`
	Module string `json:"module"`
	Param  any    `json:"param"`
}

type rpcResponse struct {
	Code    int    `json:"code"`
	Ts      int64  `json:"ts"`
	StartTs int64  `json:"start_ts"`
	TraceID string `json:"traceid"`
}

type rpcSubResponse struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
}

func (c *QQMusic) rpcCall(ctx context.Context,
	protocol string, method string, module string,
	param any,
) (json.RawMessage, error) {
	reqBody := map[string]any{protocol: rpcRequest{
		Method: method,
		Module: module,
		Param:  param,
	}}

	respBodyBuf, err := c.rpcDoRequest(ctx, reqBody)
	if err != nil {
		return nil, fmt.Errorf("qqMusicClient[rpcCall] do request: %w", err)
	}

	// check rpc response status
	respStatus := rpcResponse{}
	if err := json.Unmarshal(respBodyBuf, &respStatus); err != nil {
		return nil, fmt.Errorf("qqMusicClient[rpcCall] unmarshal response: %w", err)
	}
	if respStatus.Code != 0 {
		return nil, fmt.Errorf("qqMusicClient[rpcCall] rpc error: %d", respStatus.Code)
	}

	// parse response data
	var respBody map[string]json.RawMessage
	if err := json.Unmarshal(respBodyBuf, &respBody); err != nil {
		return nil, fmt.Errorf("qqMusicClient[rpcCall] unmarshal response: %w", err)
	}

	subRespBuf, ok := respBody[protocol]
	if !ok {
		return nil, fmt.Errorf("qqMusicClient[rpcCall] sub-response not found")
	}

	subResp := rpcSubResponse{}
	if err := json.Unmarshal(subRespBuf, &subResp); err != nil {
		return nil, fmt.Errorf("qqMusicClient[rpcCall] unmarshal sub-response: %w", err)
	}

	if subResp.Code != 0 {
		return nil, fmt.Errorf("qqMusicClient[rpcCall] sub-response error: %d", subResp.Code)
	}

	return subResp.Data, nil
}

func (c *QQMusic) downloadFile(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("qmc[downloadFile] init request: %w", err)
	}

	//req.Header.Set("Accept", "image/webp,image/*,*/*;q=0.8") // jpeg is preferred to embed in audio
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.8,en-US;q=0.6,en;q=0.5;q=0.4")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/53.0.47.134 Safari/537.36 QBCore/3.53.47.400 QQBrowser/9.0.2524.400")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("qmc[downloadFile] send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("qmc[downloadFile] unexpected http status %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func NewQQMusicClient() *QQMusic {
	return &QQMusic{
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}
