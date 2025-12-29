$Env:GOOS = 'js'
$Env:GOARCH = 'wasm'
go build -tags assert_disabled,http_enabled -o clone1_1.wasm github.com/marisvali/clone1
Remove-Item Env:GOOS
Remove-Item Env:GOARCH

$client = New-Object System.Net.WebClient
$client.Credentials = New-Object System.Net.NetworkCredential($Env:FTP_USER, $Env:FTP_PASSWORD)
$client.UploadFile("ftp://ftp.playful-patterns.com/public_html/clone1_1.wasm", "clone1_1.wasm")