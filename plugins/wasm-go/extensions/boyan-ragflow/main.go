package main

import (
	"boyan-ragflow/ragflow"
	"encoding/json"
	"fmt"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"net/http"
	"strings"
)

type AIRagConfig struct {
	RAGEngineClient                 wrapper.HttpClient
	RAGEngineAPIKey                 string
	RAGEngineDatasetIDs             []string
	RAGEngineDocumentIDs            []string
	RAGEnginePage                   int
	RAGEnginePageSize               int
	RAGEngineThreshold              float64
	RAGEngineTopK                   int
	RAGEngineRerankID               string
	RAGEngineKeyword                bool
	RAGEngineHighlight              bool
	RAGEngineEndpoint               string
	RAGEngineVectorSimilarityWeight float64
}

// ========== 主函数注册插件 ==========
func main() {
	wrapper.SetCtx(
		"ai-rag",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders),
	)
}

// ========== 配置解析函数 ==========
func parseConfig(json gjson.Result, config *AIRagConfig, log wrapper.Log) error {

	checkList := []string{
		"rag.api_key",
		"rag.endpoint",
		"rag.serviceHost",
		"rag.serviceFQDN",
		"rag.servicePort",
	}
	for _, checkEntry := range checkList {
		if !json.Get(checkEntry).Exists() {
			log.Warnf("Configuration missing: %s", checkEntry)
			return fmt.Errorf("%s not found in plugin config", checkEntry)
		}
	}
	config.RAGEngineAPIKey = json.Get("rag.api_key").String()
	config.RAGEngineEndpoint = json.Get("rag.endpoint").String()

	// 检查 dataset_ids 和 document_ids 至少有一个存在
	hasDatasetIDs := json.Get("rag.dataset_ids").Exists()
	hasDocumentIDs := json.Get("rag.document_ids").Exists()

	if !hasDatasetIDs && !hasDocumentIDs {
		log.Warnf("Either 'rag.dataset_ids' or 'rag.document_ids' must be provided")
		return fmt.Errorf("either 'rag.dataset_ids' or 'rag.document_ids' must be provided")
	}

	// 只有字段存在时才赋值 dataset_ids / document_ids
	if hasDatasetIDs {
		json.Get("rag.dataset_ids").ForEach(func(key, value gjson.Result) bool {
			config.RAGEngineDatasetIDs = append(config.RAGEngineDatasetIDs, value.String())
			return true
		})
	}

	if hasDocumentIDs {
		json.Get("rag.document_ids").ForEach(func(key, value gjson.Result) bool {
			config.RAGEngineDocumentIDs = append(config.RAGEngineDocumentIDs, value.String())
			return true
		})
	}

	//page（默认 1）
	if json.Get("rag.page").Exists() {
		config.RAGEnginePage = int(json.Get("rag.page").Int())
	} else {
		config.RAGEnginePage = 1
	}

	// page_size（默认 30）
	if json.Get("rag.page_size").Exists() {
		config.RAGEnginePageSize = int(json.Get("rag.page_size").Int())
	} else {
		config.RAGEnginePageSize = 30
	}

	// similarity_threshold（默认 0.2）
	if json.Get("rag.similarity_threshold").Exists() {
		config.RAGEngineThreshold = json.Get("rag.similarity_threshold").Float()
	} else {
		config.RAGEngineThreshold = 0.2
	}

	// vector_similarity_weight（默认 0.3）
	if json.Get("rag.vector_similarity_weight").Exists() {
		config.RAGEngineVectorSimilarityWeight = json.Get("rag.vector_similarity_weight").Float()
	} else {
		config.RAGEngineVectorSimilarityWeight = 0.3
	}

	// top_k（默认 1024）
	if json.Get("rag.top_k").Exists() {
		config.RAGEngineTopK = int(json.Get("rag.top_k").Int())
	} else {
		config.RAGEngineTopK = 1024
	}

	// rerank_id（默认 -1，表示不启用）
	if json.Get("rag.rerank_id").Exists() {
		config.RAGEngineRerankID = json.Get("rag.rerank_id").String()
	}

	// keyword（默认 false）
	if json.Get("rag.keyword").Exists() {
		config.RAGEngineKeyword = json.Get("rag.keyword").Bool()
	} else {
		config.RAGEngineKeyword = false
	}

	// highlight（默认 false）
	if json.Get("rag.highlight").Exists() {
		config.RAGEngineHighlight = json.Get("rag.highlight").Bool()
	} else {
		config.RAGEngineHighlight = false
	}

	// 构造 client 配置
	host := json.Get("rag.serviceHost").String()
	fqdn := json.Get("rag.serviceFQDN").String()
	port := json.Get("rag.servicePort").Int()

	config.RAGEngineClient = wrapper.NewClusterClient(wrapper.FQDNCluster{
		FQDN: fqdn,
		Port: port,
		Host: host,
	})

	return nil
}

// ========== 请求头处理 ==========
func onHttpRequestHeaders(wrapper.HttpContext, AIRagConfig, wrapper.Log) types.Action {
	proxywasm.RemoveHttpRequestHeader("content-length")
	return types.ActionContinue
}

// ========== 请求体处理 ==========
func onHttpRequestBody(ctx wrapper.HttpContext, config AIRagConfig, body []byte, log wrapper.Log) types.Action {

	log.Infof("Processing request body: %s", string(body))
	//proxywasm.SendHttpResponse(200, nil, []byte("张峰 是🐢"), -1)
	var rawRequest ragflow.Request
	// 解析请求体
	if err := json.Unmarshal(body, &rawRequest); err != nil {
		log.Errorf("Failed to parse request body: %v", err)
		return types.ActionContinue
	}
	// 检查 Messages 是否为空
	if len(rawRequest.Messages) == 0 {
		log.Warnf("Empty messages in request")
		return types.ActionContinue
	}

	lastMessage := rawRequest.Messages[len(rawRequest.Messages)-1]
	userQuery := lastMessage.Content
	log.Infof("User query: %s", userQuery)

	retrievalReq := &ragflow.RetrievalRequest{
		Question:               userQuery,
		DatasetIDs:             config.RAGEngineDatasetIDs,
		DocumentIDs:            config.RAGEngineDocumentIDs,
		Page:                   config.RAGEnginePage,
		PageSize:               config.RAGEnginePageSize,
		SimilarityThreshold:    config.RAGEngineThreshold,
		VectorSimilarityWeight: config.RAGEngineVectorSimilarityWeight,
		TopK:                   config.RAGEngineTopK,
		RerankID:               config.RAGEngineRerankID,
		Keyword:                config.RAGEngineKeyword,
		Highlight:              config.RAGEngineHighlight,
	}

	reqBody, _ := json.Marshal(retrievalReq)
	log.Infof("Sending retrieval request to %s with body: %s", config.RAGEngineEndpoint, string(reqBody))

	headers := [][2]string{
		{"Content-Type", "application/json"},
		{"Authorization", "Bearer " + config.RAGEngineAPIKey},
	}

	log.Infof("Retrieval request headers: %v", headers)
	log.Infof("config.RAGEngineClient: %v", config.RAGEngineClient)
	if config.RAGEngineClient != nil {
		log.Infof("config.RAGEngineClient is not nil")
	} else {
		log.Infof("config.RAGEngineClient is nil")
	}
	config.RAGEngineClient.Post(
		config.RAGEngineEndpoint,
		headers,
		reqBody,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			log.Infof("response body: %v", responseBody)
			var resp ragflow.RetrievalResponse
			if err := json.Unmarshal(responseBody, &resp); err != nil || resp.Code != 0 {
				log.Errorf("Failed to retrieve context: %v", err)
				return
			}
			log.Infof("Retrieval response data: %+v", resp.Data)

			rawRequest.Messages = rawRequest.Messages[:len(rawRequest.Messages)-1]

			traceDocs := make([]string, 0, len(resp.Data.Chunks))
			recallContents := make([]string, 0, len(resp.Data.Chunks))

			for _, chunk := range resp.Data.Chunks {
				if chunk.Similarity >= config.RAGEngineThreshold {
					recallContents = append(recallContents, chunk.Content)
					traceDocs = append(traceDocs, chunk.DocumentID)
					log.Infof("Chunk matched: %s (Similarity: %.2f)", chunk.Content, chunk.Similarity)
				}
			}
			if len(recallContents) > 0 {
				log.Infof("Adding %d retrieved chunks to request", len(recallContents))
				for _, content := range recallContents {
					rawRequest.Messages = append(rawRequest.Messages, ragflow.Message{Role: "user", Content: content})
				}
				rawRequest.Messages = append(rawRequest.Messages, ragflow.Message{Role: "user", Content: fmt.Sprintf("现在，请回答以下问题：\n%s", userQuery)})
				newBody, _ := json.Marshal(rawRequest)
				log.Infof("Modified request body: %s", string(newBody))
				proxywasm.ReplaceHttpRequestBody(newBody)

				traceStr := strings.Join(traceDocs, ", ")
				proxywasm.SetProperty([]string{"trace_span_tag.rag_docs"}, []byte(traceStr))
				ctx.SetContext("x-envoy-rag-recall", true)
			}
			proxywasm.ResumeHttpRequest()
		},
		5000,
	)
	return types.ActionPause
}

// ========== 响应头处理 ==========
func onHttpResponseHeaders(ctx wrapper.HttpContext, _ AIRagConfig, log wrapper.Log) types.Action {
	recall, ok := ctx.GetContext("x-envoy-rag-recall").(bool)
	if ok && recall {
		log.Infof("Adding 'x-envoy-rag-recall: true' to response headers")
		proxywasm.AddHttpResponseHeader("x-envoy-rag-recall", "true")
	} else {
		log.Infof("Adding 'x-envoy-rag-recall: false' to response headers")
		proxywasm.AddHttpResponseHeader("x-envoy-rag-recall", "false")
	}
	return types.ActionContinue
}
