package env

import "github.com/yaacov/kc-utils/pkg/v2v/config"

// Config is the kc-v2v runtime configuration (Forklift V2V_* env vars).
type Config = config.Config

const (
	EnvLibvirtURL                   = config.EnvLibvirtURL
	EnvInPlace                      = config.EnvInPlace
	EnvExtraArgs                    = config.EnvExtraArgs
	EnvVmName                       = config.EnvVmName
	EnvNewName                      = config.EnvNewName
	EnvRootDisk                     = config.EnvRootDisk
	EnvStaticIPs                    = config.EnvStaticIPs
	EnvSource                       = config.EnvSource
	EnvDiskPath                     = config.EnvDiskPath
	EnvFirmware                     = config.EnvFirmware
	EnvLocalMigration               = config.EnvLocalMigration
	EnvHostName                     = config.EnvHostName
	EnvNbdeClevis                   = config.EnvNbdeClevis
	EnvMultipleIPsPerNic            = config.EnvMultipleIPsPerNic
	EnvVsphereVmwareDriverRemoval   = config.EnvVsphereVmwareDriverRemoval
	EnvWindowsRegistryNetworkConfig = config.EnvWindowsRegistryNetworkConfig
	EnvWaitForGuestReboot           = config.EnvWaitForGuestReboot
	EnvOverlayEnabled               = config.EnvOverlayEnabled
	EnvFingerprint                  = config.EnvFingerprint
	EnvCopyConcurrency              = config.EnvCopyConcurrency
	EnvOffline                      = config.EnvOffline
	EnvBackend                      = config.EnvBackend

	DefaultCopyConcurrency      = config.DefaultCopyConcurrency
	DefaultCaBundle             = config.DefaultCaBundle
	DefaultCaCert               = config.DefaultCaCert
	DefaultWorkdir              = config.DefaultWorkdir
	DefaultInspectionOutputFile = config.DefaultInspectionOutputFile
	DefaultDynamicScriptsDir    = config.DefaultDynamicScriptsDir
	DefaultLuksDir              = config.DefaultLuksDir
	DefaultMountRoot            = config.DefaultMountRoot
	BlockGlob                   = config.BlockGlob
	FSGlob                      = config.FSGlob
)
