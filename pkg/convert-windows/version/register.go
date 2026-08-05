package version

func init() {
	Register("win11", win11{baseHandler{name: "win11"}})
	Register("win10", win10{baseHandler{name: "win10"}})
	Register("win81", win81{baseHandler{name: "win81"}})
	Register("win8", win8{baseHandler{name: "win8"}})
	Register("win7", win7{baseHandler{name: "win7"}})
	Register("win2008r2", win2008r2{baseHandler{name: "win2008r2"}})
	Register("win2008", win2008{baseHandler{name: "win2008"}})
	Register("winvista", winvista{baseHandler{name: "winvista"}})
	Register("win2003", win2003{baseHandler{name: "win2003"}})
	Register("winxp", winxp{baseHandler{name: "winxp"}})
}
