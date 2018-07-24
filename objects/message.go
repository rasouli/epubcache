package objects

type NotifyMsg struct {
	Etag string `json:"etag"`
	User string `json:"user"`
	File string `json:"file"`
	Path string `json:"path"`
}

type TempLinkMsg struct {
	Etag string `json:"etag"`
	User string `json:"user"`
}

type TempLinkMsgV2 struct {
	User string `json:"user"`
	File string `json:"file"`
	Hash string `json:"hash"`
}

type EncryptMsg struct {
	Hash string `json:"hash"`
	Key string  `json:"key"`
}

type MetaData struct {
	CoverFile  string            `json:"cover"`
	Attributes map[string]string `json:"attributes"`
}

type FileKeyMsg struct {
	NotifyMsg
	MetaData
	Key string `json:"key"`
}

type ShelfQueryMsg struct {
	User  string   `json:"user"`
	Etags []string `json:"etags"`
}

type ShelfQueryMsgV2 struct {
	User   string   `json:"user"`
	Files  []string `json:"files"`
	Hashes []string `json:"hashes"`
}

type ShelfQueryResponse struct {
	Message string              `json:"message"`
	Result  string              `json:"result"`
	Files   []map[string]string `json:"files"`
}

type ShelfQueryResponseV2 struct {
	Message string              `json:"message"`
	Result  string              `json:"result"`
	Files   []map[string]string `json:"files"`
	Hashes  []map[string]string `json:"hashes"`
}

type DeleteMsg struct {
	User string `json:"user"`
	Etag string `json:"etag"`
}
