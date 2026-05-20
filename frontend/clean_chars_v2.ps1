$path = "c:\Users\yusha\Documents\HOME_DATA\HONET\Project\saas-pekan\frontend\src"
$files = Get-ChildItem -Path $path -Filter *.tsx -Recurse

$utf8NoBom = New-Object System.Text.UTF8Encoding $false

foreach ($file in $files) {
    $content = [System.IO.File]::ReadAllText($file.FullName, $utf8NoBom)
    if ($content.Contains("协同")) {
        $content = $content.Replace("协同", "").TrimEnd() + "`r`n"
        [System.IO.File]::WriteAllText($file.FullName, $content, $utf8NoBom)
        Write-Host "Cleaned $($file.FullName)"
    }
}
