package gowt

import "github.com/haiodo/gowt/swt"

// Windows has no file backend of its own: the PDF is the output of the Microsoft Print to PDF driver.
const pdfPrinterRequired = true

func gcHandle(gc *swt.GC) uintptr { return uintptr(gc.Handle) }
