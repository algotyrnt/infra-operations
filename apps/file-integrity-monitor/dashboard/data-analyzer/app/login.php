<?php
/**  Copyright (c) 2026 WSO2 LLC. (https://www.wso2.com).

 * WSO2 LLC. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

 ob_start();

 /* Secure session settings - must be before session_start() */
 ini_set('session.cookie_httponly', '1');
 ini_set('session.cookie_secure', '1');
 ini_set('session.cookie_samesite', 'Strict');
 
 session_start();
 
 // Check if the form is submitted
 if ($_SERVER["REQUEST_METHOD"] == "POST") {
 
     $username = getenv('DASH_USER');
     $password = getenv('DASH_PASS');
 
     $inputUser = $_POST["username"] ?? '';
     $inputPass = $_POST["password"] ?? '';
 
     if ($username === false || $password === false || $username === '' || $password === '') {
         $error = "Login is not configured.";
 
     } elseif ($inputUser === $username && $inputPass === $password) {
 
         /* Prevent session fixation */
         session_regenerate_id(true);
 
         $_SESSION["loggedin"] = true;
 
         header("Location: dashboard.php");
         exit;
 
     } else {
         $error = "Invalid username or password.";
     }
 }
 ?>
 
 <!DOCTYPE html>
 <html lang="en">
 <head>
     <meta charset="UTF-8">
     <meta name="viewport" content="width=device-width, initial-scale=1.0">
     <title>Login</title>
 
     <style>
         body {
             font-family: Arial, sans-serif;
             background-color: #fff;
             color: #333;
             margin: 0;
             padding: 0;
             display: flex;
             justify-content: center;
             align-items: center;
             height: 100vh;
             background-color: #f7f7f7;
         }
 
         .login-container {
             background-color: #fff;
             padding: 40px;
             border-radius: 8px;
             box-shadow: 0 4px 8px rgba(0,0,0,0.1);
             width: 300px;
         }
 
         h1 {
             text-align: center;
             color: #ff7f00;
             margin-bottom: 20px;
             font-size: 24px;
         }
 
         h2 {
             text-align: center;
             color: #333;
             font-size: 18px;
         }
 
         .form-group {
             margin-bottom: 15px;
         }
 
         .form-group label {
             display: block;
             margin-bottom: 5px;
         }
 
         .form-group input {
             width: 100%;
             padding: 10px;
             border: 1px solid #ccc;
             border-radius: 4px;
         }
 
         .form-group input[type="submit"] {
             background-color: #ff7f00;
             color: #fff;
             border: none;
             cursor: pointer;
             font-weight: bold;
         }
 
         .form-group input[type="submit"]:hover {
             background-color: #e66a00;
         }
 
         .error-message {
             color: red;
             text-align: center;
             margin-bottom: 20px;
         }
 
         .powered-by {
             text-align: center;
             margin-top: 20px;
             font-size: 12px;
         }
     </style>
 </head>
 
 <body>
 
 <div class="login-container">
 
 <h1>WSO2 - FIM</h1>
 <h2>Login</h2>
 
 <?php
 if (isset($error)) {
     echo "<p class='error-message'>" . htmlspecialchars($error, ENT_QUOTES, 'UTF-8') . "</p>";
 }
 ?>
 
 <form method="post" action="login.php">
 
 <div class="form-group">
 <label for="username">Username:</label>
 <input type="text" id="username" name="username" required>
 </div>
 
 <div class="form-group">
 <label for="password">Password:</label>
 <input type="password" id="password" name="password" required>
 </div>
 
 <div class="form-group">
 <input type="submit" value="Login">
 </div>
 
 </form>
 
 <div class="powered-by">
 <p>Powered by DigiOps</p>
 </div>
 
 </div>
 
 </body>
 </html>
