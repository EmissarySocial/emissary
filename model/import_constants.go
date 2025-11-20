package model

// ImportStateNew represents an Import record that has just been created,
// but has not yet been approved by the account owner.
const ImportStateNew = "NEW"

// ImportStateDoAuthorize represents an Import record that has been given
// a source account.  This is a transient state that is set by the UX and
// will be processed by the Import service, resolving as
// "AUTHORIZING" or "AUTHORIZATION-ERROR"
const ImportStateDoAuthorize = "DO-AUTHORIZE"

// ImportStateAuthorizing represents an Import record that has been given a SourceID
// and is now waiting on an external service to authorize the migration request.
const ImportStateAuthorizing = "AUTHORIZING"

// ImportStateAuthorizationError represents an Import record that has been halted
// because of an error during the authorization process.
// A description of the error should be present in the import.Error field.
// The import will not continue until the user fixes the error.
const ImportStateAuthorizationError = "AUTHORIZATION-ERROR"

// ImportStateAuthorized represents an Import record that has been authorized
// by the remote account holder (via OAuth), and is ready for the local user
// to start running.
const ImportStateAuthorized = "AUTHORIZED"

// ImportStateDoImport represents an Import record that is starting to
// pull records from the remote server.  This is a transient state that is set
// by the UX and will be processed by the Import service, resolving to
// "IMPORTING"
const ImportStateDoImport = "DO-IMPORT"

// ImportStateImporting represents an Import record that is currently importing
// records from the remote server.  If an error is encountered during the import
// process, then the Import record will be set to "IMPORT-ERROR" state.
const ImportStateImporting = "IMPORTING"

// ImportStateImportError represents an Import record that encountered a fatal
// error while importing records. A description of the error should be present
// in the import.Error field. The import will not continue until the error
// (likely caused by the remote server) is fixed.
const ImportStateImportError = "IMPORT-ERROR"

// ImportStateReviewing represents an Import record that has completed importing
// remote records, and is being reviewed by the account owner
const ImportStateReviewing = "REVIEWING"

// ImportStateDoMove represents an Import record that is starting to announce
// to the network that records have moved from the remove server to the local
// server.  This is a transient state that is set by the UX and will be processed
// by the Import service, resolving to "MOVING"
const ImportStateDoMove = "DO-MOVE"

// ImportStateMoving represents an Import record that has is currently sending
// `Move` activities to the network to announce that records are moving from
// the remote server to the local server.  When this process exits, it will
// resolve as "MOVE-ERROR" or "DONE"
const ImportStateMoving = "MOVING"

// ImportStateMoveError represents an Import record that encountered a fatal
// error while moving records from the remote server.  A description of the
// error should be present in the import.Error field.  The import will not
// continue until the error (likely caused by the remote server) is fixed.
const ImportStateMoveError = "MOVE-ERROR"

// ImportStateDone represents an Import record that has been completed,
// and all of it's records have been moved to their new locations successfully
const ImportStateDone = "DONE"
