package helper

import . "github.com/onsi/ginkgo/v2" //nolint:staticcheck

type Team string

const (
	TeamAtlas       = "Atlas"
	TeamCabbage     = "Cabbage"
	TeamHoneybadger = "Honeybadger"
	TeamPhoenix     = "Phoenix"
	TeamRocket      = "Rocket"
	TeamShield      = "Shield"
	TeamTenet       = "Tenet"
)

var TEAM_ID = map[Team]string{
	TeamAtlas:       "S013DF1G0TU",
	TeamCabbage:     "S02FMKBLZD5",
	TeamHoneybadger: "S02G77D7GUA",
	TeamPhoenix:     "S02H54GV65R",
	TeamRocket:      "S01DAK3RRBP",
	TeamShield:      "S0419AZLVU5",
	TeamTenet:       "S07KQ7PCUSW",
}

// SetResponsibleTeam annotates the current test spec with the team that is responsible for it passing
func SetResponsibleTeam(t Team) {
	AddReportEntry("TEAM", string(t))
	AddReportEntry("TEAM_ID", TEAM_ID[t])
}
