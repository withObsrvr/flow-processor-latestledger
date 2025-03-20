// schemaapi.go
package main

// SchemaProvider is an interface for plugins that provide GraphQL schema components
type SchemaProvider interface {
	// GetSchemaDefinition returns GraphQL type definitions for this plugin
	GetSchemaDefinition() string

	// GetQueryDefinitions returns GraphQL query definitions for this plugin
	GetQueryDefinitions() string
}
