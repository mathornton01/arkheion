package handlers

// files.go — File upload/download is handled in books.go (UploadFile, DownloadFile)
// because they are tightly coupled to the book record lifecycle.
//
// This file is reserved for any future standalone file management endpoints,
// such as:
//   POST /api/v1/files/covers     - Upload a custom cover image
//   GET  /api/v1/files/covers/:id - Fetch cover image
//
// For now it serves as a documentation placeholder.
