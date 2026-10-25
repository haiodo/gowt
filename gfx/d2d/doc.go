// Package d2d replays gfx Pictures with Direct2D 1.1 and DirectWrite on Windows, with no window:
// the device is a D3D11 WARP device, so no GPU is needed and a service, CI or an SSH session in
// session 0 can render. It is empty on other systems.
//
// Experimental, like gfx: the API may change without notice.
package d2d
