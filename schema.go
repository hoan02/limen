package limen

type SchemaDefinitionMap map[SchemaName]SchemaDefinition

type Schema interface {
	GetTableName() SchemaTableName
	GetField(name SchemaField) string
	ToStorage(data Model) map[string]any
	FromStorage(data map[string]any) Model
	Serialize(data Model) map[string]any
	GetSoftDeleteField() string
	GetAdditionalFields() AdditionalFieldsFunc
	GetIDField() string
	Initialize(schemaInfo *SchemaInfo) error
	setAdditionalFields(additionalFields AdditionalFieldsFunc)
	setModelTransformer(transformer ModelTransformer)
}

type Model interface {
	// Raw returns the model raw data as returned from the database
	Raw() map[string]any
}

// ModelTransformer transforms a model into its JSON response representation.
type ModelTransformer func(model Model) map[string]any

// ModelTransformers maps logical schema names to model transformers.
type ModelTransformers map[SchemaName]ModelTransformer

type BaseSchema struct {
	// A function to return a map of additional fields to be added to the schema when creating a record. e.g:
	//  func(ctx context.Context) map[string]any {
	// 		return map[string]any{
	//  		"uuid": uuid.New().String(),
	//  		"created_at": time.Now(),
	//  		"updated_at": time.Now(),
	// 		 }
	//	 }
	// NOTE: fields here will override the global additional fields function.
	additionalFields AdditionalFieldsFunc

	// schemaInfo contains all resolved schema information including table name, field mappings, and resolver
	schemaInfo *SchemaInfo

	// A function to serialize the model to a json object for returning to the client
	Serializer ModelTransformer
}

func (b *BaseSchema) GetTableName() SchemaTableName {
	if b.schemaInfo == nil {
		return ""
	}
	return b.schemaInfo.tableName
}

func (b *BaseSchema) setAdditionalFields(additionalFields AdditionalFieldsFunc) {
	b.additionalFields = additionalFields
}

func (b *BaseSchema) setModelTransformer(transformer ModelTransformer) {
	b.Serializer = transformer
}

func (b *BaseSchema) GetAdditionalFields() AdditionalFieldsFunc {
	return b.additionalFields
}

func (b *BaseSchema) GetIDField() string {
	return b.GetField(SchemaIDField)
}

func (b *BaseSchema) GetSoftDeleteField() string {
	return b.GetField(SchemaSoftDeleteField)
}

func (b *BaseSchema) GetFieldResolver() *SchemaResolver {
	if b.schemaInfo == nil {
		return nil
	}
	return b.schemaInfo.resolver
}

func (b *BaseSchema) GetField(name SchemaField) string {
	if b.schemaInfo == nil {
		return ""
	}
	return b.schemaInfo.GetField(name)
}

func (b *BaseSchema) Serialize(data Model) map[string]any {
	if b.Serializer != nil {
		return b.Serializer(data)
	}
	return data.Raw()
}

func (b *BaseSchema) Initialize(schemaInfo *SchemaInfo) error {
	b.schemaInfo = schemaInfo
	return nil
}
