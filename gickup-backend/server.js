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
  const logFilePath = path.join(__dirname, 'var', 'logs', 'gickup.log');

  fs.readFile(logFilePath, 'utf8', (err, data) => {
    if (err) {
      console.error('Error reading log file:', err);
      res.status(500).json({ error: 'Error reading log file' });
      return;
    }

    const logEntries = data.trim().split('\n').map((line) => {
      try {
        const parsedEntry = JSON.parse(line);
        if (parsedEntry && parsedEntry.level === 'info' && parsedEntry.message === 'Backup run complete') {
          const duration = parseFloat(parsedEntry.duration.replace('s', ''));
          if (!isNaN(duration)) {
            parsedEntry.duration = duration; // Add duration to the log entry
            return parsedEntry;
          }
        }
      } catch (error) {
        console.error('Error parsing log entry:', error);
      }
      return null; // Ignore malformed or incomplete log entries
    }).filter(entry => entry !== null); // Remove null entries

    const successfulRuns = logEntries.length; // Count of successful runs

    // Calculate total duration and individual durations
    const totalDuration = logEntries.reduce((acc, entry) => {
      acc.total += entry.duration;
      acc.individualDurations.push(entry.duration);
      return acc;
    }, { total: 0, individualDurations: [] });

    // Respond with the processed backup statistics
    res.json({ backupData: { successfulRuns, totalDuration, individualDurations: totalDuration.individualDurations } });
  });
});

const yaml = require('js-yaml');

app.get('/api/configFiles', (req, res) => {
  const configFolder = path.join(__dirname);
  const yamlFiles = [];

  fs.readdir(configFolder, (err, files) => {
    if (err) {
      console.error('Error reading config files:', err);
      res.status(500).json({ error: 'Error reading config files' });
      return;
    }

    files.forEach(fileName => {
      if (fileName.endsWith('.yml')) {
        const filePath = path.join(configFolder, fileName);
        try {
          const fileContent = fs.readFileSync(filePath, 'utf-8');
          const parsedYAML = yaml.load(fileContent); 
          if (parsedYAML && parsedYAML.source && parsedYAML.destination) {
            const source = Object.keys(parsedYAML.source)[0]; 
            const destination = Object.keys(parsedYAML.destination)[0]; 
            yamlFiles.push({ fileName: fileName, source, destination });
          } else {
            console.error(`Error parsing YAML file ${fileName}: Invalid format`);
          }
        } catch (error) {
          console.error(`Error reading/parsing YAML file ${fileName}:`, error);
        }
      }
    });

    res.json({ files: yamlFiles });
  });
});


// Start the server
const PORT = 5000;
app.listen(PORT, () => {
  console.log(`Server is running on port ${PORT}`);
});
