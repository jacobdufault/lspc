// Copyright 2018 Jacob Dufault
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"github.com/mailru/easyjson"
)

//go:generate easyjson -all

type LsDocumentURI string

type LsPosition struct {
	// Note: these are 0-based.
	Line      int `json:"line"`
	Character int `json:"character"`
}

type LsRange struct {
	Start LsPosition `json:"start"`
	End   LsPosition `json:"end"`
}

type LsLocation struct {
	URI   LsDocumentURI `json:"uri"`
	Range LsRange       `json:"range"`
}

type LsSymbolKind int

const (
	Unknown       LsSymbolKind = 0
	File          LsSymbolKind = 1
	Module        LsSymbolKind = 2
	Namespace     LsSymbolKind = 3
	Package       LsSymbolKind = 4
	Class         LsSymbolKind = 5
	Method        LsSymbolKind = 6
	Property      LsSymbolKind = 7
	Field         LsSymbolKind = 8
	Constructor   LsSymbolKind = 9
	Enum          LsSymbolKind = 10
	Interface     LsSymbolKind = 11
	Function      LsSymbolKind = 12
	Variable      LsSymbolKind = 13
	Constant      LsSymbolKind = 14
	String        LsSymbolKind = 15
	Number        LsSymbolKind = 16
	Boolean       LsSymbolKind = 17
	Array         LsSymbolKind = 18
	Object        LsSymbolKind = 19
	Key           LsSymbolKind = 20
	Null          LsSymbolKind = 21
	EnumMember    LsSymbolKind = 22
	Struct        LsSymbolKind = 23
	Event         LsSymbolKind = 24
	Operator      LsSymbolKind = 25
	TypeParameter LsSymbolKind = 26
)

type LsTextDocumentIdentifier struct {
	URI LsDocumentURI `json:"uri"`
}

type LsVersionedTextDocumentIdentifier struct {
	URI LsDocumentURI `json:"uri"`
	// Version number of this document.  number | null
	Version *int `json:"version"`
}

type LsTextDocumentPositionParams struct {
	// The text document.
	TextDocument LsTextDocumentIdentifier `json:"textDocument"`
	// The position inside the text document.
	Position LsPosition `json:"position"`
}

type LsTextEdit struct {
	// The range of the text document to be manipulated. To insert
	// text into a document create a range where start === end.
	Range LsRange `json:"range"`

	// The string to be inserted. For delete operations use an
	// empty string.
	NewText string `json:"newText"`
}

type LsTextDocumentItem struct {
	// The text document's URI.
	URI LsDocumentURI `json:"uri"`

	// The text document's language identifier.
	LanguageID string `json:"languageId"`

	// The version number of this document (it will strictly increase after each
	// change, including undo/redo).
	Version int `json:"version"`

	// The content of the opened text document.
	Text string `json:"text"`
}

type LsInitializeParams struct {
	/**
	 * The process Id of the parent process that started
	 * the server. Is null if the process has not been started by another process.
	 * If the parent process is not alive then the server should exit (see exit notification) its process.
	 */
	// processId number | null;

	/**
	 * The rootPath of the workspace. Is null
	 * if no folder is open.
	 *
	 * @deprecated in favour of rootUri.
	 */
	// rootPath?: string | null;

	/**
	 * The rootUri of the workspace. Is null if no
	 * folder is open. If both `rootPath` and `rootUri` are set
	 * `rootUri` wins.
	 */
	RootURI LsDocumentURI `json:"rootUri"`

	/**
	 * User provided initialization options.
	 */
	InitializationOptions easyjson.RawMessage `json:"initializationOptions"`

	/**
	 * The capabilities provided by the client (editor or tool)
	 */
	// capabilities: ClientCapabilities;

	/**
	 * The initial trace setting. If omitted trace is disabled ('off').
	 */
	// trace?: 'off' | 'messages' | 'verbose';

	/**
	 * The workspace folders configured in the client when the server starts.
	 * This property is only available if the client supports workspace folders.
	 * It can be `null` if the client supports workspace folders but none are
	 * configured.
	 *
	 * Since 3.6.0
	 */
	// workspaceFolders?: WorkspaceFolder[] | null;
}

// RequestID is the id of a request/response
type RequestID int

// JSONRPCHeader is used to identify a message.
type JSONRPCHeader struct {
	JSONRPC string              `json:"jsonrpc"` // Should be "2.0"
	Method  string              `json:"method"`  // ie, "textDocument/codeLens"
	ID      RequestID           `json:"id"`
	Params  easyjson.RawMessage `json:"params"`
}

// NotificationInitialized is sent from the server to the client after the
// client is ready to go.
type NotificationInitialized struct{}

/*

struct lsResponseError {
  enum class lsErrorCodes : int {
    ParseError = -32700,
    InvalidRequest = -32600,
    MethodNotFound = -32601,
    InvalidParams = -32602,
    InternalError = -32603,
    serverErrorStart = -32099,
    serverErrorEnd = -32000,
    ServerNotInitialized = -32002,
    UnknownErrorCode = -32001,
    RequestCancelled = -32800,
  };

  lsErrorCodes code;
  // Short description.
  std::string message;

  void Write(Writer& visitor);
};


// cquery extension
struct lsLocationEx : lsLocation {
  optional<std::string_view> containerName;
  optional<lsSymbolKind> parentKind;
  // Avoid circular dependency on symbol.h
  optional<uint16_t> role;
};
MAKE_REFLECT_STRUCT(lsLocationEx, uri, range, containerName, parentKind, role);

template <typename T>
struct lsCommand {
  // Title of the command (ie, 'save')
  std::string title;
  // Actual command identifier.
  std::string command;
  // Arguments to run the command with.
  // **NOTE** This must be serialized as an array. Use
  // MAKE_REFLECT_STRUCT_WRITER_AS_ARRAY.
  T arguments;
};

template <typename TData, typename TCommandArguments>
struct lsCodeLens {
  // The range in which this code lens is valid. Should only span a single line.
  lsRange range;
  // The command this code lens represents.
  optional<lsCommand<TCommandArguments>> command;
  // A data entry field that is preserved on a code lens item between
  // a code lens and a code lens resolve request.
  TData data;
};

struct lsTextDocumentEdit {
  // The text document to change.
  lsVersionedTextDocumentIdentifier textDocument;

  // The edits to be applied.
  std::vector<lsTextEdit> edits;
};
MAKE_REFLECT_STRUCT(lsTextDocumentEdit, textDocument, edits);

struct lsWorkspaceEdit {
  // Holds changes to existing resources.
  // changes ? : { [uri:string]: TextEdit[]; };
  // std::unordered_map<lsDocumentUri, std::vector<lsTextEdit>> changes;

  // An array of `TextDocumentEdit`s to express changes to specific a specific
  // version of a text document. Whether a client supports versioned document
  // edits is expressed via `WorkspaceClientCapabilites.versionedWorkspaceEdit`.
  std::vector<lsTextDocumentEdit> documentChanges;
};
MAKE_REFLECT_STRUCT(lsWorkspaceEdit, documentChanges);

struct lsFormattingOptions {
  // Size of a tab in spaces.
  int tabSize;
  // Prefer spaces over tabs.
  bool insertSpaces;
};
MAKE_REFLECT_STRUCT(lsFormattingOptions, tabSize, insertSpaces);

// MarkedString can be used to render human readable text. It is either a
// markdown string or a code-block that provides a language and a code snippet.
// The language identifier is sematically equal to the optional language
// identifier in fenced code blocks in GitHub issues. See
// https://help.github.com/articles/creating-and-highlighting-code-blocks/#syntax-highlighting
//
// The pair of a language and a value is an equivalent to markdown:
// ```${language}
// ${value}
// ```
//
// Note that markdown strings will be sanitized - that means html will be
// escaped.
struct lsMarkedString {
  optional<std::string> language;
  std::string value;
};
void Reflect(Writer& visitor, lsMarkedString& value);

struct lsTextDocumentContentChangeEvent {
  // The range of the document that changed.
  optional<lsRange> range;
  // The length of the range that got replaced.
  optional<int> rangeLength;
  // The new text of the range/document.
  std::string text;
};
MAKE_REFLECT_STRUCT(lsTextDocumentContentChangeEvent, range, rangeLength, text);

struct lsTextDocumentDidChangeParams {
  lsVersionedTextDocumentIdentifier textDocument;
  std::vector<lsTextDocumentContentChangeEvent> contentChanges;
};
MAKE_REFLECT_STRUCT(lsTextDocumentDidChangeParams,
                    textDocument,
                    contentChanges);

// Show a message to the user.
enum class lsMessageType : int { Error = 1, Warning = 2, Info = 3, Log = 4 };
MAKE_REFLECT_TYPE_PROXY(lsMessageType)
struct Out_ShowLogMessageParams {
  lsMessageType type = lsMessageType::Error;
  std::string message;
};
MAKE_REFLECT_STRUCT(Out_ShowLogMessageParams, type, message);
struct Out_ShowLogMessage : public lsOutMessage<Out_ShowLogMessage> {
  enum class DisplayType { Show, Log };
  DisplayType display_type = DisplayType::Show;

  std::string method();
  Out_ShowLogMessageParams params;
};
template <typename TVisitor>
void Reflect(TVisitor& visitor, Out_ShowLogMessage& value) {
  REFLECT_MEMBER_START();
  REFLECT_MEMBER(jsonrpc);
  std::string method = value.method();
  REFLECT_MEMBER2("method", method);
  REFLECT_MEMBER(params);
  REFLECT_MEMBER_END();
}

struct Out_LocationList : public lsOutMessage<Out_LocationList> {
  lsRequestId id;
  std::vector<lsLocationEx> result;
};
MAKE_REFLECT_STRUCT(Out_LocationList, jsonrpc, id, result);


*/
