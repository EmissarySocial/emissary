package config

// StorageTypeMongo represents a configuration database stored in a MongoDB database
const StorageTypeMongo = "MONGODB"

// StorageTypeFile represents a configuration database stored in a JSON file
const StorageTypeFile = "FILE"

// ConfigSourceCommandLine represents that the config file location was specified via the "--config" command line argument
const ConfigSourceCommandLine = "COMMAND"

// ConfigSourceEnvironment represents that the config file location was specified via the "EMISSARY_CONFIG" environment variable
const ConfigSourceEnvironment = "ENVIRONMENT"

// ConfigSourceDefault represents that the config file location was not specified, so the default value of "file://./config.json" was used
const ConfigSourceDefault = "DEFAULT"

// DefaultConfigDatabase is the MongoDB database that holds the server configuration when neither
// the --db flag, the EMISSARY_CONFIG_DB environment variable, nor the connection string names one
const DefaultConfigDatabase = "emissary"

// DefaultConfigCollection is the MongoDB collection that holds the server configuration when
// neither the --collection flag nor the EMISSARY_CONFIG_COLLECTION environment variable names one
const DefaultConfigCollection = "config"
