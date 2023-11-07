import React, { useState, useEffect } from 'react';
import axios from 'axios';
import { List, ListItem, ListItemText, Typography, Button, Box } from '@mui/material';

function JobsList() {
  const [jobFiles, setJobFiles] = useState([]);
  const [error, setError] = useState(null); 

  useEffect(() => {
    axios.get('http://localhost:5000/api/configFiles')
      .then(response => {
        setJobFiles(response.data.files); 
      })
      .catch(error => {
        console.error('Error fetching job files:', error);
        setError('Error fetching job files. Please try again later.'); 
      });
  }, []);

  const handleRunNow = (jobName) => {
    axios.post('http://localhost:5000/api/runGoApp', { fileName: jobName, runNow: true })
      .then(response => {
        console.log(`Job ${jobName} has been triggered successfully with --runnow flag.`);
      })
      .catch(error => {
        console.error(`Error triggering job ${jobName}:`, error);
      });
  };

  return (
    <Box p={3} bgcolor="#f5f5f5" borderRadius={4} boxShadow={3}>
      <Typography variant="h4" mb={3}>
        Jobs
      </Typography>
      {error && <Typography variant="body1" color="error" mb={2}>{error}</Typography>} 
      <List>
        {jobFiles.map((fileData, index) => (
          <ListItem key={index} sx={{ borderRadius: 2, bgcolor: 'white', mb: 2, p: 2, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <ListItemText primary={fileData.fileName} secondary={`Source: ${JSON.stringify(fileData.source)}, Destination: ${JSON.stringify(fileData.destination)}`} />
            <Button variant="contained" color="primary" onClick={() => handleRunNow(fileData.fileName)}>
              Run Now
            </Button>
          </ListItem>
        ))}
      </List>
    </Box>
  );
}

export default JobsList;
