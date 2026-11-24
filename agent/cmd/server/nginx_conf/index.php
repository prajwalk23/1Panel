<?php
error_reporting(E_ALL);
ini_set('display_errors', 1);

echo '<h1 style="text-align: center;">Welcome to PHP!</h1>';
echo '<h2>Version info</h2>';

echo '<ul>';
echo '<li>PHP version: ', PHP_VERSION, '</li>';
echo '</ul>';

echo '<h2>Installed extensions</h2>';
printExtensions();

/**
 * Get the list of installed extensions
 */
function printExtensions()
{
    echo '<ol>';
    foreach (get_loaded_extensions() as $i => $name) {
        echo "<li>", $name, '=', phpversion($name), '</li>';
    }
    echo '</ol>';
}