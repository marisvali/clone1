<!DOCTYPE html>
<html>
<body>

<h1>Clone1 collection script</h1>

<?php
$servername = "172.232.206.74";
$username = "playfulp_temp";
$password = "comeonthough";
$dbname = "playfulp_clone1";

function LogInfo($message) {
 	file_put_contents("./submit-playthrough-clone1.log", "INFO: " . $message . "\n", FILE_APPEND);
}

function LogError($message) {
	file_put_contents("./submit-playthrough-clone1.log", "ERROR: " . $message . "\n", FILE_APPEND);
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
        LogInfo("We got user: " . $user);
        $release_version = $_POST['release_version'];
        LogInfo("We got release_version: " . $release_version);
        $simulation_version = $_POST['simulation_version'];
        LogInfo("We got simulation_version: " . $simulation_version);
        $input_version = $_POST['input_version'];
        LogInfo("We got input_version: " . $input_version);
        $id = $_POST['id'];
        LogInfo("We got id: " . $id);
        if (isset($_FILES['playthrough'])) {
            $file = $_FILES['playthrough'];
            LogInfo("Found file.");
            $fileName = $file['name'];
            LogInfo("We got file name: " . $fileName);
            $fileTmpPath = $file['tmp_name'];
            LogInfo("We got file tmp path: " . $fileTmpPath);
            $fileContent = mysqli_real_escape_string($conn, file_get_contents($fileTmpPath));
            LogInfo("Read the file contents!");
            
            $sql = "UPDATE playthroughs SET end_moment=now(), playthrough = '$fileContent' WHERE user = '$user' AND id = '$id'";
        } else {
            $sql = "INSERT INTO playthroughs(start_moment, user, release_version, simulation_version, input_version, id) " .
            "VALUES (now(), '$user', '$release_version', '$simulation_version', '$input_version', '$id')";
        }
        
        try {
            $conn->query($sql);
            LogInfo("Data successfully inserted!");
        } catch(Exception $e) {
            LogError("Error inserting data: " . $e->getMessage());
        }

        $conn->close();
    }
}
LogInfo("End.");
?>

</body>
</html>