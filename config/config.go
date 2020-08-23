package config

import (
	"github.com/superdentist/superdentist-backend/constants"
	"github.com/superdentist/superdentist-backend/env"
	"github.com/superdentist/superdentist-backend/global"
)

// Init ....
// read config variables from environmental variables or '.env' file and
// populate globals.Options with the values
func Init() {
	// Load '.env' from current working directory.
	//env.Load(".env")
	global.Options.Debug = env.Bool(constants.ENV_DEBUG, global.Options.Debug)

}
