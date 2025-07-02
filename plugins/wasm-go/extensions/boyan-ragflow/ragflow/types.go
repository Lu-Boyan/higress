package ragflow

// RetrievalRequest 用于构建检索请求的结构体
type RetrievalRequest struct {
	Question               string   `json:"question"`
	DatasetIDs             []string `json:"dataset_ids,omitempty"`
	DocumentIDs            []string `json:"document_ids,omitempty"`
	Page                   int      `json:"page,omitempty"`
	PageSize               int      `json:"page_size,omitempty"`
	SimilarityThreshold    float64  `json:"similarity_threshold,omitempty"`
	VectorSimilarityWeight float64  `json:"vector_similarity_weight,omitempty"`
	TopK                   int      `json:"top_k,omitempty"`
	RerankID               string   `json:"rerank_id"`
	Keyword                bool     `json:"keyword,omitempty"`
	Highlight              bool     `json:"highlight,omitempty"`
}

type RetrievalResponse struct {
	Code int          `json:"code"`
	Data ResponseData `json:"data"`
}

type ResponseData struct {
	Chunks  []Chunk  `json:"chunks"`
	DocAggs []DocAgg `json:"doc_aggs"`
	Total   int      `json:"total"`
}

type Chunk struct {
	ID                string   `json:"id"`
	DocumentID        string   `json:"document_id"`
	Content           string   `json:"content"`
	ContentLTks       string   `json:"content_ltks"`
	Highlight         string   `json:"highlight"`
	ImageID           string   `json:"image_id"`
	ImportantKeywords []string `json:"important_keywords"`
	KBID              string   `json:"kb_id"`
	Positions         []string `json:"positions"`
	Similarity        float64  `json:"similarity"`
	TermSimilarity    float64  `json:"term_similarity"`
	VectorSimilarity  float64  `json:"vector_similarity"`
	DocumentKeyword   string   `json:"document_keyword"`
}

type DocAgg struct {
	Count   int    `json:"count"`
	DocID   string `json:"doc_id"`
	DocName string `json:"doc_name"`
}

type Request struct {
	Model            string    `json:"model"`
	Messages         []Message `json:"messages"`
	FrequencyPenalty float64   `json:"frequency_penalty"`
	PresencePenalty  float64   `json:"presence_penalty"`
	Stream           bool      `json:"stream"`
	Temperature      float64   `json:"temperature"`
	Topp             float64   `json:"top_p"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
