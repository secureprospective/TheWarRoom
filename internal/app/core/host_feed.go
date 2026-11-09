package core

import "context"

func (s *State) Ready() error { <-s.Started; return s.StartupErr }

// FeedContext is the app lifetime context MFL reads run under; Background before startup.
func (s *State) FeedContext() context.Context {
	if s.Ctx == nil {
		return context.Background()
	}
	return s.Ctx
}

func (s *State) LogFallback(line string) { s.FallbackLog(line) }

// MFLDestination is the MFL league host and season that links and exports point at.
func (s *State) MFLDestination() (string, int) {
	host := ""
	if s.MFLClient != nil {
		host = s.MFLClient.LeagueHost()
	}
	return host, s.Season
}

func (s *State) OpenBrowser(target string) { s.OpenURL(s.Ctx, target) }
