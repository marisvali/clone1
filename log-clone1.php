<?php
$servername = "172.232.206.74";
$username = "playfulp_temp";
$password = "comeonthough";
$dbname = "playfulp_clone1";

function LogInfo($message) {
	// file_put_contents("./log-clone1.log", "INFO: " . $message . "\n", FILE_APPEND);
}

function LogError($message) {
	file_put_contents("./log-clone1.log", "ERROR: " . $message . "\n", FILE_APPEND);
    http_response_code(513);
	die();
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
        $release_version = $_POST['release_version'];
        $simulation_version = $_POST['simulation_version'];
        $input_version = $_POST['input_version'];
        $id = $_POST['id'];
        $level = $_POST['level'];
        $message = $_POST['message'];
        $file = $_FILES['playthrough'];
        $fileName = $file['name'];
        $fileTmpPath = $file['tmp_name'];
        $fileContent = mysqli_real_escape_string($conn, file_get_contents($fileTmpPath));

        $sql = "INSERT INTO logs(moment, user, release_version, simulation_version, input_version, id, level, message, playthrough) " .
                            "VALUES (now(), '$user', '$release_version', '$simulation_version', '$input_version', '$id', '$level', '$message', '$fileContent')";
        try {
            $conn->query($sql);
            LogInfo("INSERT successful!");
        } catch(Exception $e) {
            LogError("Error executing INSERT: " . $e->getMessage());
        }

        $conn->close();
    }
}
LogInfo("End.");
?>