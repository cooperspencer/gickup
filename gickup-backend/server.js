const express = require('express');
const bodyParser = require('body-parser');
const app = express();
const fs = require('fs');
const path = require('path');
const cors = require('cors');
const AnsiToHtml = require('ansi-to-html');
const ansiToHtml = new AnsiToHtml();

app.use(bodyParser.json());
app.use(cors());
app.use(express.json());


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

const { spawn } = require('child_process');

app.post('/api/runGoApp', (req, res) => {
  const { fileName, runNow } = req.body;
  const goAppPath = path.join(__dirname, '..', 'main.go');
  const configFilePath = path.join(__dirname, fileName);

  let command = ['run', goAppPath, configFilePath];

  if (runNow) {
    command.push('--runnow');
  }

  console.log('Executing command:', command.join(' '));

  
  const logStream = fs.createWriteStream(path.join(__dirname, 'history', 'run.log'), { flags: 'a' });

  const goProcess = spawn('go', command);

  goProcess.stdout.on('data', (data) => {
    const lines = data.toString().split('\n');
    lines.forEach(line => {
      if (line.includes('INF')) {
        console.log('INF:', line); 
      } else if (line.includes('ERR')) {
        console.error('ERR:', line); 
      }
    });

    
    logStream.write(data);

    res.write(data); 
  });

  goProcess.stderr.on('data', (data) => {
    console.error('ERROR:', data.toString()); 

    
    logStream.write(data);

    res.write(data); 
  });

  goProcess.on('close', (code) => {
    console.log(`Go process exited with code ${code}`);
    res.end(); 

   
    logStream.end();
  });
});

app.get('/api/fetchLogFile', (req, res) => {
  const logFilePath = path.join(__dirname, 'history', 'run.log');

  try {
    const data = fs.readFileSync(logFilePath, 'utf8');
    const formattedData = ansiToHtml.toHtml(data); 
    console.log('unfomated data', data)
    console.log('fomated data' , formattedData )
    res.send(formattedData);
  } catch (err) {
    console.error('Error reading log file:', err);
    res.status(500).send('Error reading log file');
  }
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
            parsedEntry.duration = duration; 
            return parsedEntry;
          }
        }
      } catch (error) {
        console.error('Error parsing log entry:', error);
      }
      return null; 
    }).filter(entry => entry !== null);

    const successfulRuns = logEntries.length; 

    
    const totalDuration = logEntries.reduce((acc, entry) => {
      acc.total += entry.duration;
      acc.individualDurations.push(entry.duration);
      return acc;
    }, { total: 0, individualDurations: [] });

    
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

const baseDirectory = '/backups';

app.get('/api/files', (req, res) => {
  const requestedPath = req.query.path || '';
  const directoryPath = path.join(baseDirectory, requestedPath);

  try {
    const result = readDirectoryRecursive(directoryPath);
    res.json(result);
  } catch (error) {
    console.error('Error reading directory:', error);
    res.status(500).json({ error: 'Internal Server Error' });
  }
});

function readDirectoryRecursive(directoryPath) {
  const files = fs.readdirSync(directoryPath);
  const result = {
    files: [],
    folders: {}
  };

  files.forEach(file => {
    const filePath = path.join(directoryPath, file);
    const isDirectory = fs.statSync(filePath).isDirectory();
    if (isDirectory) {
      const subdirectoryContents = readDirectoryRecursive(filePath);
      result.folders[file] = subdirectoryContents;
    } else {
      result.files.push(file);
    }
  });

  return result;
}

// Start the server
const PORT = 5000;
app.listen(PORT, () => {
  console.log(`Server is running on port ${PORT}`);
});
