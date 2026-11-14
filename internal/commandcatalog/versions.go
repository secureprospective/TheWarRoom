package catalog

// VersionsReading reconciles a caller's cache without disclosing ledger positions.
type VersionsReading struct {
	ServerEpoch    string            `json:"serverEpoch"`
	AuthGeneration string            `json:"authGeneration"`
	League         string            `json:"league"`
	Season         int               `json:"season"`
	Topics         map[string]string `json:"topics"`
	Football       string            `json:"football"`
	Live           string            `json:"live"`
	Values         string            `json:"values"`
	ServerTime     string            `json:"serverTime"`
	NextBoundary   string            `json:"nextBoundary"`
}
