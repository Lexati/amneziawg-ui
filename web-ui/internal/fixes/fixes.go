// Package fixes holds the workarounds for upstream bugs (fyne and its wasm
// driver), kept out of the UI code: each one lives in its own file, together
// with a description of the bug and the condition under which it can be
// dropped again.
package fixes

// Install puts every workaround in place; call it before the window starts.
func Install() {
	installNonASCIIKeyboard()
}
