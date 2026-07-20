package apis

import "fmt"

// EmitLink prints a download URL for CLI use. No-op when Mute/Quiet so library
// callers and concurrent probes can rely on returned strings without stdout races.
func EmitLink(link string) {
	if link == "" || MuteMode || QuietMode {
		return
	}
	fmt.Println(link)
}
