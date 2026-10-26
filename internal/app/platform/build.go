package platform

func BuildLabel(version, commit string) string {
	if commit == "" {
		return version
	}
	return version + " (" + commit + ")"
}
