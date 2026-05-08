// service.go contains systemd-related helpers for the discovery package:
// ignored unit lists, cgroup parsing for unit name extraction, and systemd
// status checking.
package discovery

// ignoredSystemdUnits is the set of systemd unit names whose processes
// should be excluded from discovery entirely. These are OS-level system
// services and cloud provider agents, not user applications.
var ignoredSystemdUnits = map[string]bool{
	// Desktop / GUI
	"plasma-plasmashell": true,
	"gnome-shell":        true,
	"gnome-terminal":     true,
	"gnome-initial-setup": true,
	"gnome-software":     true,
	"xfce4-session":      true,
	"dbus":               true,
	"systemd":            true,
	"init":               true,
	"orca":               true,

	// Debian/Ubuntu system services
	"networkd-dispatcher":         true,
	"unattended-upgrades":         true,
	"unattended-upgrade-shutdown": true,
	"apport":                      true,
	"aptd":                        true,
	"update-notifier":             true,
	"update-manager":              true,
	"motd-news":                   true,
	"apt-daily":                   true,
	"apt-daily-upgrade":           true,
	"landscape-client":            true,
	"ubuntu-advantage":            true,

	// RHEL/Fedora/Rocky/Alma system services
	"firewalld":            true,
	"tuned":                true,
	"dnf-automatic":        true,
	"dnf-makecache":        true,
	"yum-cron":             true,
	"subscription-manager": true,
	"rhsmcertd":            true,
	"insights-client":      true,
	"abrt":                 true,
	"abrt-ccpp":            true,
	"abrt-oops":            true,

	// Cross-distro system services
	"packagekit":  true,
	"fail2ban":    true,
	"certbot":     true,
	"fwupd":       true,

	// Cloud provider agents
	"cloud-init":                  true,
	"cloud-config":                true,
	"cloud-final":                 true,
	"walinuxagent":                true,
	"waagent":                     true,
	"google-accounts-daemon":      true,
	"google-network-daemon":       true,
	"google-clock-skew-daemon":    true,
	"google-ip-forwarding-daemon": true,
	"cfn-hup":                     true,
}
