$path = "c:\Users\yusha\Documents\HOME_DATA\HONET\Project\PEKAN\frontend\src"
$files = Get-ChildItem -Path $path -Filter *.tsx -Recurse
$files += Get-ChildItem -Path $path -Filter *.ts -Recurse

foreach ($file in $files) {
    # Read raw bytes
    $bytes = [System.IO.File]::ReadAllBytes($file.FullName)
    
    $isUtf16Le = $false
    
    # Check for BOM
    if ($bytes.Length -ge 2 -and $bytes[0] -eq 255 -and $bytes[1] -eq 254) {
        $isUtf16Le = $true
    }
    # Check for NUL bytes indicating UTF-16LE without BOM
    elseif ($bytes -contains 0) {
        $isUtf16Le = $true
    }
    
    if ($isUtf16Le) {
        $text = [System.IO.File]::ReadAllText($file.FullName, [System.Text.Encoding]::Unicode)
        $utf8NoBom = New-Object System.Text.UTF8Encoding $false
        [System.IO.File]::WriteAllText($file.FullName, $text, $utf8NoBom)
        Write-Host "Converted $($file.FullName)"
    }
}
