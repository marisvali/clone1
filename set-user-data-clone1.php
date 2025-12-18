<?php
$servername = "172.232.206.74";
$username = "playfulp_temp";
$password = "comeonthough";
$dbname = "playfulp_clone1";

function LogInfo($message) {
	// file_put_contents("./set-user-data.log", "INFO: " . $message . "\n", FILE_APPEND);
}

LogInfo("Start.");
if ($_SERVER['REQUEST_METHOD'] == 'POST') {
    LogInfo("Attempt to connect to database.");

    $conn = new mysqli($servername, $username, $password, $dbname);
    if ($conn->connect_error) {
        LogInfo("Connection failed: " . $conn->connect_error);
    } else {
        LogInfo("Connection succeeded!");
        
        $user = $_POST['user'];
        $data = $_POST['data'];
        LogInfo("We got user: " . $user);
        LogInfo("We got data: " . $data);
        $sql = "REPLACE INTO user_data (user, data) VALUES ('$user', '$data')";
        
        try {
            $conn->query($sql);
            LogInfo("REPLACE successful!");
        } catch(Exception $e) {
            LogError("Error executing REPLACE: " . $e->getMessage());
        }
        
        $conn->close();
    }
}
LogInfo("End.");
?>