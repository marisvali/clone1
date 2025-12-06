<?php
$servername = "172.232.206.74";
$username = "playfulp_temp";
$password = "comeonthough";
$dbname = "playfulp_clone1";

function LogInfo($message) {
	file_put_contents("./get-user-data-clone1.log", "INFO: " . $message . "\n", FILE_APPEND);
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
        LogInfo("We got user: " . $user);
        $sql = "SELECT data FROM user_data WHERE user = '$user'";
        
        try {
            $result = $conn->query($sql);
            LogInfo("Query successful!");
        } catch(Exception $e) {
            LogError("Error querying: " . $e->getMessage());
        }
        
        if ($result->num_rows > 0) {
            $row = $result->fetch_assoc();
            echo $row["data"];
        } else {
            echo "";
        }
        $conn->close();
    }
}
LogInfo("End.");
?>