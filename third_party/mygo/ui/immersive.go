package ui

// SetImmersive configures whether the window's root lays out edge-to-edge
// under mobile system bars rather than padding the root with top/bottom safe
// area insets. Views that enable immersive mode should position header controls
// and bottom navigation relative to Context.SafeArea.
func SetImmersive(c *Context, on bool) {
	if rt := c.runtime(); rt != nil {
		rt.immersive = on
	}
}
