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
