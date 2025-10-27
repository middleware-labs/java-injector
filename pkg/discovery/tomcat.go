package discovery

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
)

// TomcatInfo contains information about a Tomcat deployment
type TomcatInfo struct {
	IsTomcat      bool
	CatalinaBase  string
	CatalinaHome  string
	InstanceName  string
	Webapps       []string
	ServerXMLPath string
	Port          int
}

// detectTomcatDeployment checks if this is a Tomcat process and extracts info
func (d *discoverer) detectTomcatDeployment(javaProc *JavaProcess, cmdArgs []string) *TomcatInfo {
	tomcatInfo := &TomcatInfo{}

	// Check if this is Tomcat by looking for catalina or Bootstrap
	isTomcat := false
	for _, arg := range cmdArgs {
		argLower := strings.ToLower(arg)
		if strings.Contains(argLower, "catalina") ||
			strings.Contains(argLower, "org.apache.catalina.startup.bootstrap") ||
			strings.Contains(argLower, "tomcat") {
			isTomcat = true
			break
		}
	}

	if !isTomcat {
		return tomcatInfo
	}

	tomcatInfo.IsTomcat = true

	// Extract CATALINA_BASE and CATALINA_HOME
	for _, arg := range cmdArgs {
		if strings.HasPrefix(arg, "-Dcatalina.base=") {
			tomcatInfo.CatalinaBase = strings.TrimPrefix(arg, "-Dcatalina.base=")
		}
		if strings.HasPrefix(arg, "-Dcatalina.home=") {
			tomcatInfo.CatalinaHome = strings.TrimPrefix(arg, "-Dcatalina.home=")
		}
	}

	// Extract instance name from CATALINA_BASE path
	if tomcatInfo.CatalinaBase != "" {
		parts := strings.Split(tomcatInfo.CatalinaBase, "/")
		if len(parts) > 0 {
			tomcatInfo.InstanceName = parts[len(parts)-1]
		}
	}

	// Discover webapps
	if tomcatInfo.CatalinaBase != "" {
		webapps := d.discoverTomcatWebapps(tomcatInfo.CatalinaBase)
		tomcatInfo.Webapps = webapps
	}

	// Get server.xml path
	if tomcatInfo.CatalinaBase != "" {
		tomcatInfo.ServerXMLPath = filepath.Join(tomcatInfo.CatalinaBase, "conf", "server.xml")

		// Try to extract port from server.xml
		port := d.extractTomcatPort(tomcatInfo.ServerXMLPath)
		if port > 0 {
			tomcatInfo.Port = port
		}
	}

	return tomcatInfo
}

// discoverTomcatWebapps discovers deployed webapps in a Tomcat instance
func (d *discoverer) discoverTomcatWebapps(catalinaBase string) []string {
	webappsDir := filepath.Join(catalinaBase, "webapps")

	dirs, err := ioutil.ReadDir(webappsDir)
	if err != nil {
		return []string{}
	}

	var webapps []string
	for _, dir := range dirs {
		if dir.IsDir() {
			// Skip ROOT - it's typically the default webapp
			if dir.Name() != "ROOT" && dir.Name() != "manager" && dir.Name() != "host-manager" {
				webapps = append(webapps, dir.Name())
			}
		} else if strings.HasSuffix(dir.Name(), ".war") {
			// Also include WAR files
			webappName := strings.TrimSuffix(dir.Name(), ".war")
			if webappName != "ROOT" {
				webapps = append(webapps, webappName)
			}
		}
	}

	return webapps
}

// extractTomcatPort tries to extract the HTTP port from server.xml
func (d *discoverer) extractTomcatPort(serverXMLPath string) int {
	content, err := ioutil.ReadFile(serverXMLPath)
	if err != nil {
		return 0
	}

	// Look for <Connector port="8080" protocol="HTTP/1.1"
	re := regexp.MustCompile(`<Connector[^>]*port="(\d+)"[^>]*protocol="HTTP`)
	matches := re.FindSubmatch(content)

	if len(matches) > 1 {
		var port int
		fmt.Sscanf(string(matches[1]), "%d", &port)
		return port
	}

	return 0
}

// ExtractTomcatInfo is a public method to get Tomcat information
func (jp *JavaProcess) ExtractTomcatInfo() *TomcatInfo {
	d := &discoverer{}
	return d.detectTomcatDeployment(jp, jp.ProcessCommandArgs)
}

// IsTomcat checks if this is a Tomcat process
func (jp *JavaProcess) IsTomcat() bool {
	tomcatInfo := jp.ExtractTomcatInfo()
	return tomcatInfo.IsTomcat
}

// GetTomcatWebapps returns the list of deployed webapps
func (jp *JavaProcess) GetTomcatWebapps() []string {
	tomcatInfo := jp.ExtractTomcatInfo()
	return tomcatInfo.Webapps
}
