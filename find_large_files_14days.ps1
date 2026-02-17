# Find large files created in last 14 days on C: drive
$twoWeeksAgo = (Get-Date).AddDays(-14)
$minSize = 100MB

$results = Get-ChildItem -Path C:\ -File -Recurse -ErrorAction SilentlyContinue | 
    Where-Object { 
        $_.Length -gt $minSize -and 
        $_.CreationTime -gt $twoWeeksAgo 
    } | 
    Select-Object FullName, 
        @{Name='SizeMB'; Expression={[math]::Round($_.Length/1MB, 2)}}, 
        CreationTime | 
    Sort-Object CreationTime -Descending

Write-Host "`n========================================" 
Write-Host "Large Files on C: (>100MB, Last 14 Days)"
Write-Host "========================================`n"

$counter = 1
foreach ($file in $results) {
    $daysDiff = ((Get-Date) - $file.CreationTime).Days
    $dayLabel = switch ($daysDiff) {
        0 { "Today" }
        1 { "Yesterday" }
        default { "$daysDiff days ago" }
    }
    
    Write-Host "[$counter] $($file.FullName)"
    Write-Host "    Size: $($file.SizeMB) MB"
    Write-Host "    Created: $($file.CreationTime) ($dayLabel)"
    Write-Host ""
    $counter++
}

Write-Host "========================================"
Write-Host "Total files found: $($results.Count)"
Write-Host "========================================"