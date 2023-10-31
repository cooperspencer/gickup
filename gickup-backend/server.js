const express = require('express');
const bodyParser = require('body-parser');
const app = express();
const fs = require('fs');
const path = require('path');
const cors = require('cors');


app.use(bodyParser.json());
app.use(cors());


app.post('/api/saveConfiguration', (req, res) => {
  const { yamlConfig, fileName } = req.body;


  const filePath = path.join(__dirname, fileName );
  fs.writeFile(filePath, yamlConfig, 'utf8', (err) => {
    if (err) {
      console.error('Error writing configuration file:', err);
      res.status(500).json({ error: 'Error writing configuration file' });
    } else {
      console.log('Configuration saved to file:', filePath);
      res.json({ success: true });
    }
  });
});

app.post('/api/runGoApp', (req, res) => {
  const { exec } = require('child_process');
  const { fileName } = req.body;
  const goAppPath = path.join(__dirname, '..', 'main.go'); 
  const configFilePath = path.join(__dirname, fileName);
  
  const command = `"${goAppPath}" "${configFilePath}"`;
  console.log('Executing command:', command);
  exec(command, (error, stdout, stderr) => {
    if (error) {
      console.error('Error executing Go app:', error);
      res.status(500).json({ error: 'Error executing Go app' });
    } else {
      console.log('Executing command:', command);
      console.log('Go app executed successfully');
      res.json({ success: true });
    }
  });
});


app.get('/api/backupStatistics', (req, res) => {
  const logFilePath = path.join(__dirname, 'var', 'logs', 'gickup.log'); // Corrected log file path

  fs.readFile(logFilePath, 'utf8', (err, data) => {
    if (err) {
      console.error('Error reading log file:', err);
      res.status(500).json({ error: 'Error reading log file' });
      return;
    }

    const logEntries = data.trim().split('\n').map((line) => {
      try {
        return JSON.parse(line);
      } catch (error) {
        console.error('Error parsing log entry:', error);
        return null; // Ignore malformed log entries
      }
    });

    // Process backup statistics
    const successfulRuns = logEntries.filter(
      (entry) => entry && entry.level === 'info' && entry.message === 'Backup run complete'
    ).length;

    const totalDuration = logEntries.reduce((acc, entry) => {
      if (entry && entry.duration) {
        const duration = parseFloat(entry.duration.replace('s', ''));
        if (!isNaN(duration)) {
          acc += duration;
        }
      }
      return acc;
    }, 0);

    // Respond with the processed backup statistics
    res.json({ backupStatistics: { successfulRuns, totalDuration } });
  });
});

// Start the server
const PORT = 5000;
app.listen(PORT, () => {
  console.log(`Server is running on port ${PORT}`);
});
