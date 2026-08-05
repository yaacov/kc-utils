package routecleanup

import (
	"github.com/yaacov/kc-utils/pkg/convert-windows/firstboot"
	"github.com/yaacov/kc-utils/pkg/convert-windows/version"
)

type Plugin struct{}

func init() {
	firstboot.Contributors.Register("routecleanup", &Plugin{})
}

func (p *Plugin) Priority() int { return 2600 }
func (p *Plugin) Name() string  { return "remove-duplicate-routes" }

func (p *Plugin) ShouldRun(cfg *firstboot.ContributorConfig) bool {
	if cfg.Offline || len(cfg.StaticIPs) == 0 {
		return false
	}
	if cfg.Version != nil {
		if cfg.Version.StaticIPMode() == version.StaticIPWMINetsh {
			return false
		}
		if !cfg.Version.SupportsPowerShell() {
			return false
		}
	}
	return true
}

func (p *Plugin) UsesBatch(_ *firstboot.ContributorConfig) bool { return false }

func (p *Plugin) Generate(cfg *firstboot.ContributorConfig) (string, error) {
	if cfg.Options.WindowsRegistryNetwork {
		return registryScript(), nil
	}
	return defaultScript(), nil
}

func defaultScript() string {
	return `# Remove duplicate persistent routes after IP reconfiguration
$ErrorActionPreference = 'SilentlyContinue'

# Helper: remove a persistent route using Remove-NetRoute and route.exe fallback
function Remove-PersistentRoute($route) {
    try {
        Remove-NetRoute -DestinationPrefix $route.DestinationPrefix -NextHop $route.NextHop ` + "`" + `
            -InterfaceIndex $route.InterfaceIndex -PolicyStore PersistentStore -Confirm:$false -ErrorAction Stop
    } catch {
        $parts = $route.DestinationPrefix.Split("/")
        $dest = $parts[0]
        $mask = ConvertTo-SubnetMask ([int]$parts[1])
        & route delete $dest mask $mask $route.NextHop 2>&1 | Out-Null
    }
}

function ConvertTo-SubnetMask([int]$prefix) {
    $masks = @('0.0.0.0','128.0.0.0','192.0.0.0','224.0.0.0','240.0.0.0',
        '248.0.0.0','252.0.0.0','254.0.0.0','255.0.0.0','255.128.0.0',
        '255.192.0.0','255.224.0.0','255.240.0.0','255.248.0.0','255.252.0.0',
        '255.254.0.0','255.255.0.0','255.255.128.0','255.255.192.0','255.255.224.0',
        '255.255.240.0','255.255.248.0','255.255.252.0','255.255.254.0','255.255.255.0',
        '255.255.255.128','255.255.255.192','255.255.255.224','255.255.255.240',
        '255.255.255.248','255.255.255.252','255.255.255.254','255.255.255.255')
    if ($prefix -ge 0 -and $prefix -le 32) { return $masks[$prefix] }
    return '255.255.255.0'
}

try {
    $routes = Get-NetRoute -PolicyStore PersistentStore -ErrorAction Stop
} catch {
    exit 0
}

if (-not $routes) { exit 0 }

# Remove duplicate routes (same destination+gateway on different interfaces)
$grouped = $routes | Group-Object { "$($_.DestinationPrefix)-$($_.NextHop)" } | Where-Object { $_.Count -gt 1 }
$liveAdapters = Get-NetAdapter -ErrorAction SilentlyContinue

foreach ($group in $grouped) {
    $dupRoutes = $group.Group
    $isDefaultGw = $group.Name.Trim().StartsWith("0.0.0.0/0-")

    if ($isDefaultGw) {
        # Keep the route on a live adapter with the lowest metric
        $sorted = $dupRoutes | Sort-Object RouteMetric
        $kept = $false
        foreach ($route in $sorted) {
            $adapterAlive = $liveAdapters | Where-Object { $_.InterfaceIndex -eq $route.InterfaceIndex }
            if ($adapterAlive -and -not $kept) {
                $kept = $true
                continue
            }
            Remove-PersistentRoute $route
        }
        if (-not $kept -and $sorted.Count -gt 0) {
            # Re-add the best route
            $best = $sorted[0]
            New-NetRoute -DestinationPrefix $best.DestinationPrefix -NextHop $best.NextHop ` + "`" + `
                -InterfaceIndex $best.InterfaceIndex -PolicyStore PersistentStore -ErrorAction SilentlyContinue
        }
    } else {
        # Non-gateway duplicates: remove all, re-add one on a live interface
        $bestRoute = $null
        foreach ($route in $dupRoutes) {
            $adapterAlive = $liveAdapters | Where-Object { $_.InterfaceIndex -eq $route.InterfaceIndex }
            if ($adapterAlive -and -not $bestRoute) {
                $bestRoute = $route
            }
            Remove-PersistentRoute $route
        }
        if ($bestRoute) {
            New-NetRoute -DestinationPrefix $bestRoute.DestinationPrefix -NextHop $bestRoute.NextHop ` + "`" + `
                -InterfaceIndex $bestRoute.InterfaceIndex -PolicyStore PersistentStore -ErrorAction SilentlyContinue
        }
    }
}

# Remove stale routes on dead interfaces
$allPersistent = Get-NetRoute -PolicyStore PersistentStore -AddressFamily IPv4 -ErrorAction SilentlyContinue
$activeIndexes = @(Get-NetAdapter -IncludeHidden -ErrorAction SilentlyContinue | ForEach-Object { $_.InterfaceIndex })
if ($activeIndexes.Count -gt 0) {
    foreach ($route in $allPersistent) {
        if ($activeIndexes -notcontains $route.InterfaceIndex) {
            Remove-PersistentRoute $route
        }
    }
}
` + "\r\n"
}

func registryScript() string {
	return `# Remove duplicate persistent routes after IP reconfiguration (registry mode)
$ErrorActionPreference = 'SilentlyContinue'

function Remove-PersistentRoute($route) {
    try {
        Remove-NetRoute -DestinationPrefix $route.DestinationPrefix -NextHop $route.NextHop ` + "`" + `
            -InterfaceIndex $route.InterfaceIndex -PolicyStore PersistentStore -Confirm:$false -ErrorAction Stop
    } catch {
        $parts = $route.DestinationPrefix.Split("/")
        $dest = $parts[0]
        $mask = ConvertTo-SubnetMask ([int]$parts[1])
        & route delete $dest mask $mask $route.NextHop 2>&1 | Out-Null
    }
}

function ConvertTo-SubnetMask([int]$prefix) {
    $masks = @('0.0.0.0','128.0.0.0','192.0.0.0','224.0.0.0','240.0.0.0',
        '248.0.0.0','252.0.0.0','254.0.0.0','255.0.0.0','255.128.0.0',
        '255.192.0.0','255.224.0.0','255.240.0.0','255.248.0.0','255.252.0.0',
        '255.254.0.0','255.255.0.0','255.255.128.0','255.255.192.0','255.255.224.0',
        '255.255.240.0','255.255.248.0','255.255.252.0','255.255.254.0','255.255.255.0',
        '255.255.255.128','255.255.255.192','255.255.255.224','255.255.255.240',
        '255.255.255.248','255.255.255.252','255.255.255.254','255.255.255.255')
    if ($prefix -ge 0 -and $prefix -le 32) { return $masks[$prefix] }
    return '255.255.255.0'
}

try {
    $routes = Get-NetRoute -PolicyStore PersistentStore -ErrorAction Stop
} catch {
    exit 0
}

if (-not $routes) { exit 0 }

# Remove duplicate routes (same destination+gateway, ignoring interface index and metric)
$grouped = $routes | Group-Object { "$($_.DestinationPrefix)-$($_.NextHop)" } | Where-Object { $_.Count -gt 1 }
$liveAdapters = Get-NetAdapter -ErrorAction SilentlyContinue

foreach ($group in $grouped) {
    $dupRoutes = $group.Group
    $isDefaultGw = $group.Name.Trim().StartsWith("0.0.0.0/0-")

    if ($isDefaultGw) {
        $sorted = $dupRoutes | Sort-Object RouteMetric
        $kept = $false
        foreach ($route in $sorted) {
            $adapterAlive = $liveAdapters | Where-Object { $_.InterfaceIndex -eq $route.InterfaceIndex }
            if ($adapterAlive -and -not $kept) {
                $kept = $true
                continue
            }
            Remove-PersistentRoute $route
        }
        if (-not $kept -and $sorted.Count -gt 0) {
            $best = $sorted[0]
            # Also clean PersistentRoutes registry key
            $persistKey = "HKLM:\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\PersistentRoutes"
            if (Test-Path $persistKey) {
                $nextHop = $best.NextHop
                $props = Get-Item -Path $persistKey | Select-Object -ExpandProperty Property
                foreach ($p in $props) {
                    $routePrefix = "0.0.0.0,0.0.0.0,$nextHop,"
                    if ($p.StartsWith($routePrefix)) {
                        Remove-ItemProperty -Path $persistKey -Name $p -ErrorAction SilentlyContinue
                    }
                }
            }
            New-NetRoute -DestinationPrefix $best.DestinationPrefix -NextHop $best.NextHop ` + "`" + `
                -InterfaceIndex $best.InterfaceIndex -PolicyStore PersistentStore -ErrorAction SilentlyContinue
        }
    } else {
        $bestRoute = $null
        foreach ($route in $dupRoutes) {
            $adapterAlive = $liveAdapters | Where-Object { $_.InterfaceIndex -eq $route.InterfaceIndex }
            if ($adapterAlive -and -not $bestRoute) {
                $bestRoute = $route
            }
            Remove-PersistentRoute $route
        }
        if ($bestRoute) {
            New-NetRoute -DestinationPrefix $bestRoute.DestinationPrefix -NextHop $bestRoute.NextHop ` + "`" + `
                -InterfaceIndex $bestRoute.InterfaceIndex -PolicyStore PersistentStore -ErrorAction SilentlyContinue
        }
    }
}

# Remove stale routes on dead interfaces
$allPersistent = Get-NetRoute -PolicyStore PersistentStore -AddressFamily IPv4 -ErrorAction SilentlyContinue
$activeIndexes = @(Get-NetAdapter -IncludeHidden -ErrorAction SilentlyContinue | ForEach-Object { $_.InterfaceIndex })
if ($activeIndexes.Count -gt 0) {
    foreach ($route in $allPersistent) {
        if ($activeIndexes -notcontains $route.InterfaceIndex) {
            Remove-PersistentRoute $route
        }
    }
}
` + "\r\n"
}
