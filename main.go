package main

import (
	"./icon"
	"github.com/romualdr/systray"
	"sync"
)

type Service struct {
	// name of the service
	name      string
	// tooltip for the submenu
	tooltip   string
	// path to the executable - without virtual drive
	path      string
	// executable
	exe       string
	// arguments
	args      string
	// Displayed
	windowed  bool
	isRunning bool
	open      string
	entry     *systray.MenuItem
}


var services = [4]Service{{
	name:          "Apache",
	tooltip:       "Web Server",
	path:           `\.sys\apache2\bin`,
	exe:           `httpd.exe`,
	args:          "",
	windowed:      false,
	isRunning:     false,
	open:          `http://localhost`,
}, {
	name:          "MongoDB",
	tooltip:       "Document oriented database",
	path:           `\.sys\mongodb`,
	exe:           `mongod.exe`,
	args:          `--journal --dbpath \.sys\mongodb\data --config \.sys\mongodb\mongod.yaml`,
	windowed:      false,
	isRunning:     false,
}, {
	name:          "MySQL",
	tooltip:       "Relational database",
	path:           `\.sys\mysql\bin`,
	exe:           `mysqld.exe`,
	args:          `--defaults-file=\.sys\mysql\my.ini`,
	windowed:      false,
	isRunning:     false,
}, {
	name:          "Memcached",
	tooltip:       "Cache",
	path:           `\.sys\memcached`,
	exe:           `memcached.exe`,
	args:          `-m 512`,
	windowed:      false,
	isRunning:     false,
}}
var configurations = [5]Service{{
	name:          "Apache",
	tooltip:       "Edit Apache Configuration",
	path:          `\.sys\apache2\conf\httpd.conf`,
}, {
	name:          "Virtual Hosts",
	tooltip:       "Edit Apache vHost Configuration",
	path:          `\.sys\apache2\conf\vhost.conf`,
}, {
	name:          "MySQL",
	tooltip:       "Edit MySQL configuration",
	path:          `\.sys\mysql\my.ini`,
}, {
	name:          "PHP",
	tooltip:       "Edit PHP configuration",
	path:          `\.sys\php\php.ini`,
}, {
	name:          "MongoDB",
	tooltip:       "Edit PHP configuration",
	path:          `\.sys\mongodb\mongod.yaml`,
}}
var logs = [5]Service{{
	name:          "Access logs",
	tooltip:       "Display access logs",
	path:          `\.sys\apache2\logs\access.log`,
}, {
	name:          "Error logs",
	tooltip:       "Display error logs",
	path:          `\.sys\apache2\logs\error.log`,
}, {
	name:          "MongoDB logs",
	tooltip:       "Display Mongo Logs",
	path:          `\.sys\mongodb\logs\mongod.log`,
}, {
	name:          "MySQL logs",
	tooltip:       "Display MySQL Logs",
	path:          `\.sys\mysql\logs\mysql.log`,
}, {
	name:          "MySQL errors",
	tooltip:       "Display MySQL errors",
	path:          `\.sys\mysql\logs\mysql.err`,
}}
var tools = [5]Service{{
	name:          "Adminer",
	tooltip:       "Manage MySQL",
	path:          `http://localhost/adminer`,
}, {
	name:          "APC",
	tooltip:       "APC",
	path:          `http://localhost/apc`,
}, {
	name:          "MongoDB",
	tooltip:       "MongoDB",
	path:          `http://localhost/mongodb`,
}, {
	name:          "PHP Info",
	tooltip:       "PHP Info",
	path:          `http://localhost/phpinfo`,
}, {
	name:          "Memcached",
	tooltip:       "Memcached",
	path:          `http://localhost/memcached`,
}}
var documentations = [6]Service{{
	name:          "Apache",
	tooltip:       "Web Server",
	path:          `https://httpd.apache.org/docs/2.4`,
}, {
	name:          "HTML5",
	tooltip:       "HTML documentation",
	path:          `https://developer.mozilla.org/docs/Web/HTML`,
}, {
	name:          "Javascript",
	tooltip:       "JS documentation",
	path:          `https://developer.mozilla.org/docs/Web/Javascript`,
}, {
	name:          "MongoDB",
	tooltip:       "MongoDB documentation",
	path:          `https://docs.mongodb.com/`,
}, {
	name:          "MySQL",
	tooltip:       "MySQL documentation",
	path:          `https://dev.mysql.com/doc/`,
}, {
	name:          "PHP",
	tooltip:       "PHP documentation",
	path:          `https://www.php.net/docs.php`,
}}
var started = false
var mStartStop *systray.MenuItem

func main() {
	initialize()
	systray.Run(createUI, func() {
		// Do not use - it is not called for some reasons (even when launched normally)
	})
}

func initialize() {
	if !hasStringValue("Path") {
		setStringValue("Path", getPath())
	}
	// Clean previous run if needed
	cleanup()
	createDrive()
	addToPath(vDisk + `\.sys\php`)
	addToPath(vDisk + `\.sys\miniperl`)
}

func cleanup() {
	if hasStringValue("Path") {
		setPath(getStringValue("Path"))
	}
	stopAll()
	if hasStringValue("vDisk") {
		vDisk := getStringValue("vDisk")
		if vDiskExists(vDisk) {
			removeDrive(vDisk)
		}
		setStringValue("vDisk", "")
	}
}

func start(service *Service) {
	if service.isRunning { return }
	println("Starting " + service.exe)
	execute(service.path + `\` + service.exe, service.args, service.path, service.windowed)
	setIntegerValue(service.exe, 1)
	service.isRunning = true
	service.entry.Check()
	addToPath(vDisk + `\` + service.path)
}

func stop(service *Service, wg *sync.WaitGroup) {
	if !hasIntegerValue(service.exe) {
		return
	}
	pid := getIntegerValue(service.exe)
	if pid == 0 { return }

	println("Stopping " + service.exe)
	if wg != nil {
		terminateAll(wg, service.exe)
	} else {
		terminate(service.exe)
	}
	removePath(vDisk + `\` + service.path)
	setIntegerValue(service.exe, 0)
	service.isRunning = false
	if service.entry != nil {
		service.entry.Uncheck()
	}
}

func stopAll() {
	var wg sync.WaitGroup
	for x := range services {
		stop(&services[x], &wg)
	}
	wg.Wait()
}

func updateUI() {
	hasStartedService := false
	for x := range services {
		if services[x].isRunning {
			hasStartedService = true
		}
	}
	if hasStartedService {
		mStartStop.SetTitle("Stop")
		started = true
	} else {
		mStartStop.SetTitle("Start")
		started = false
	}
}

func createUI() {
	systray.SetIcon(icon.Data)
	systray.SetTitle("XWAMP")
	systray.SetTooltip("XWAMP")

	mRestart := systray.AddMenuItem("Restart", "Restart")
	go func() {
		for {
			<-mRestart.ClickedCh
			for x := range services {
				if services[x].isRunning {
					stopAll()
					start(&services[x])
				}
			}
			updateUI()
		}
	}()

	mStartStop = systray.AddMenuItem("Start", "Start")
	go func() {
		for {
			<-mStartStop.ClickedCh
			if started == true {
				stopAll()
				started = false
			} else {
				for x := range services {
					if !services[x].isRunning {
						start(&services[x])
					}
				}
			}
			updateUI()
		}
	}()
	systray.AddSeparator()

	mServicesMenu := systray.AddMenuItem("Services", "Services")
	for x := range services {
		menuEntry := mServicesMenu.AddSubMenuItem(services[x].name, services[x].tooltip)
		services[x].entry = menuEntry
		go func(service *Service) {
			for {
				<-menuEntry.ClickedCh
				if service.isRunning {
					stop(service, nil)
				} else {
					start(service)
					if len(service.open) > 0 {
						open(service.open)
					}
				}
				updateUI()
			}
		}(&services[x])
	}

	mConfigurationsMenu := systray.AddMenuItem("Configurations", "Configurations")
	for x := range configurations {
		menuEntry := mConfigurationsMenu.AddSubMenuItem(configurations[x].name, configurations[x].tooltip)
		configurations[x].entry = menuEntry
		go func(service *Service) {
			for {
				<-menuEntry.ClickedCh
				openInEditor(service.path)
			}
		}(&configurations[x])
	}

	mLogsMenu := systray.AddMenuItem("Logs", "Logs")
	for x := range logs {
		menuEntry := mLogsMenu.AddSubMenuItem(logs[x].name, logs[x].tooltip)
		logs[x].entry = menuEntry
		go func(service *Service) {
			for {
				<-menuEntry.ClickedCh
				openInEditor(service.path)
			}
		}(&logs[x])
	}


	mDocumentationMenu := systray.AddMenuItem("Documentations", "Documentations")
	for x := range documentations {
		menuEntry := mDocumentationMenu.AddSubMenuItem(documentations[x].name, documentations[x].tooltip)
		documentations[x].entry = menuEntry
		go func(service *Service) {
			for {
				<-menuEntry.ClickedCh
				open(service.path)
			}
		}(&documentations[x])
	}

	mTools := systray.AddMenuItem("Tools", "Tools")
	for x := range tools {
		menuEntry := mTools.AddSubMenuItem(tools[x].name, tools[x].tooltip)
		tools[x].entry = menuEntry
		go func(service *Service) {
			for {
				<-menuEntry.ClickedCh
				open(service.path)
			}
		}(&tools[x])
	}

	systray.AddSeparator()
	mAbout := systray.AddMenuItem("About", "About")
	go func() {
		for {
			<-mAbout.ClickedCh
			open("https://github.com/romualdr/xwamp")
		}
	}()

	systray.AddSeparator()
	// Quit the application
	mQuitOrig := systray.AddMenuItem("Quit", "Close XWAMP")
	go func() {
		<-mQuitOrig.ClickedCh
		cleanup()
		systray.Quit()
	}()
	setCurrentDirectory(vDisk)
}
