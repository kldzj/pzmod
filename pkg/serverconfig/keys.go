package serverconfig

// Project Zomboid servertest.ini keys that pzmod reads or writes.
const (
	KeyName        = "PublicName"
	KeyDescription = "PublicDescription"
	KeyPublic      = "Public"
	KeyPassword    = "Password"
	KeyMaxPlayers  = "MaxPlayers"
	KeyMods        = "Mods"
	KeyWorkshop    = "WorkshopItems"
	KeyMap         = "Map"
)

// listSep is the canonical separator pzmod writes for Mods/WorkshopItems/Map.
const listSep = ";"
