const express = require('express');
const bodyParser = require('body-parser');
const app = express();
const fs = require('fs');
const path = require('path');

app.use(bodyParser.json());

// Define your API endpoints here
app.post('/api/saveConfiguration', (req, res) => {
  const configurationData = req.body;
  // Handle saving the configuration data here (to a file, database, etc.)
  console.log('Configuration saved:', configurationData);
  res.json({ success: true });
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
