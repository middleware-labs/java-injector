// types.go defines core types shared across the discovery package: the Language
// enum used to classify processes, and the IntegrationInspector/IntegrationRegistry
// for future host-level integration detection (e.g. Redis, Kafka).
package discovery

// Language represents a programming language detected in a process.
type Language string

const (
	LangJava   Language = "java"
	LangNode   Language = "node"
	LangPython Language = "python"
	LangGo     Language = "go"
	LangRust   Language = "rust"
	LangRuby   Language = "ruby"
	LangPHP    Language = "php"
)

