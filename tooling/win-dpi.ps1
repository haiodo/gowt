# Sets or reads the display scale of the first monitor: win-dpi.ps1 get | win-dpi.ps1 set <percent> (100, 125, ..., 500).
# Runs in the logged-on desktop session (vm.sh gui); applies at once, no sign-out. Prints "current=<percent> recommended=<percent>".
param([string]$Cmd = "get", [int]$Percent = 100)
Add-Type @"
using System; using System.Runtime.InteropServices;
public class Dpi {
 [DllImport("user32.dll")] public static extern int GetDisplayConfigBufferSizes(uint f, out uint np, out uint nm);
 [DllImport("user32.dll")] public static extern int QueryDisplayConfig(uint f, ref uint np, IntPtr paths, ref uint nm, IntPtr modes, IntPtr topo);
 [DllImport("user32.dll")] public static extern int DisplayConfigGetDeviceInfo(IntPtr p);
 [DllImport("user32.dll")] public static extern int DisplayConfigSetDeviceInfo(IntPtr p);
}
"@
$steps = 100, 125, 150, 175, 200, 225, 250, 300, 350, 400, 450, 500
$m = [Runtime.InteropServices.Marshal]
[uint32]$np = 0; [uint32]$nm = 0
[void][Dpi]::GetDisplayConfigBufferSizes(2, [ref]$np, [ref]$nm) # QDC_ONLY_ACTIVE_PATHS
$paths = $m::AllocHGlobal(72 * $np)
$modes = $m::AllocHGlobal(64 * $nm)
if ([Dpi]::QueryDisplayConfig(2, [ref]$np, $paths, [ref]$nm, $modes, [IntPtr]::Zero) -ne 0) { throw "QueryDisplayConfig failed" }
$p = $m::AllocHGlobal(32)
function Header($type, $size) {
	$m::WriteInt32($p, 0, $type); $m::WriteInt32($p, 4, $size)
	$m::WriteInt32($p, 8, $m::ReadInt32($paths, 0)); $m::WriteInt32($p, 12, $m::ReadInt32($paths, 4))
	$m::WriteInt32($p, 16, $m::ReadInt32($paths, 8))
}
Header -3 32
if ([Dpi]::DisplayConfigGetDeviceInfo($p) -ne 0) { throw "get failed" }
$rec = -$m::ReadInt32($p, 20)
$cur = $rec + $m::ReadInt32($p, 24)
if ($Cmd -eq "set") {
	$want = [Array]::IndexOf($steps, $Percent)
	if ($want -lt 0) { throw "not a scale step: $Percent" }
	Header -4 24
	$m::WriteInt32($p, 20, $want - $rec)
	if ([Dpi]::DisplayConfigSetDeviceInfo($p) -ne 0) { throw "set failed" }
	$cur = $want
}
"current=$($steps[$cur]) recommended=$($steps[$rec])"
