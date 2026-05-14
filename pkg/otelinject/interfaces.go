// interfaces.go defines the core types for the otelinject package.
// Language is a type alias for discovery.Language, kept for backward compatibility
// with mw-agent which imports otelinject.LanguageJava etc.
package otelinject

import "github.com/middleware-labs/java-injector/pkg/discovery"

// Language is an alias for discovery.Language so that mw-agent can continue
// using otelinject.Language without breaking. Because this is a type alias (=),
// values are interchangeable with discovery.Language.
type Language = discovery.Language

const (
	LanguagePython Language = "python"
	LanguageJava   Language = "java"
	LanguageNode   Language = "node"
	LanguageGo     Language = "go"
	LanguageRust   Language = "rust"
	LanguagePHP    Language = "php"
	LanguageRuby   Language = "ruby"
)

