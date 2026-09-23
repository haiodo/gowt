// OS.java now translates for real (tooling/port.sh stage 2+3): its own <Type>_sizeof() natives
// (emitStaticNativeMethod's struct-sizeof special case, in cocoa_os.go) took over from the
// stand-ins this file used to provide. Kept empty rather than deleted - see AGENTS.md on
// removing hand-written *_manual.go files.
package cocoa
