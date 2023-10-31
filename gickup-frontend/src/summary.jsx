import React from 'react';
import ReactDOM from 'react-dom';
import Typography from '@mui/material/Typography';
import Button from '@mui/material/Button';
import Box from '@mui/material/Box';
import Paper from '@mui/material/Paper';
import jsYaml from 'js-yaml';

const Summary = () => {
  const step1Data = JSON.parse(localStorage.getItem('Step1')) || {};
  const step2Data = JSON.parse(localStorage.getItem('Step2')) || {};
  const step3Data = JSON.parse(localStorage.getItem('Step3')) || {};
  const step4Data = JSON.parse(localStorage.getItem('Step4')) || {};

  const finish = () => {
    const configData = {
      '# Commented YAML Lines': [
        ...(step1Data.name ? [`# name: ${step1Data.name}`] : []),
        ...(step1Data.description ? [`# description: ${step1Data.description}`] : []),
      ],
      source: {},
      destination: {},
      cron: step4Data.cronExpression,
    };
  
    if (step2Data.selectedSource) {
      configData.source[step2Data.selectedSource] = [
        {
          ...(step2Data.token ? { token: step2Data.token } : {}),
          ...(step2Data.user ? { user: step2Data.user } : {}),
          ...(step2Data.ssh ? { ssh: step2Data.ssh } : {}),
          ...(step2Data.sshkey ? { sshkey: step2Data.sshkey } : {}),
          ...(step2Data.excluderepos ? { excluderepos: step2Data.excluderepos.split(',') } : {}),
          ...(step2Data.excludeorgs ? { excludeorgs: step2Data.excludeorgs.split(',') } : {}),
          ...(step2Data.includerepos ? { includerepos: step2Data.includerepos.split(',') } : {}),
          ...(step2Data.includeorgs ? { includeorgs: step2Data.includeorgs.split(',') } : {}),
          ...(step2Data.wiki ? { wiki: step2Data.wiki } : {}),
          ...(step2Data.starred ? { starred: step2Data.starred } : {}),
          filter: {
            ...(step2Data.filterstars ? { stars: step2Data.filterstars } : {}),
            ...(step2Data.lastactivity ? { lastactivity: step2Data.lastactivity } : {}),
            ...(step2Data.excludearchived ? { excludearchived: step2Data.excludearchived } : {}),
            ...(step2Data.languages ? { languages: step2Data.languages.split(',') } : {}),
            ...(step2Data.excludeforks ? { excludeforks: step2Data.excludeforks } : {}),
          },
        }
      ];
    }
  
    if (step3Data.SelectedDestination) {
      configData.destination[step3Data.SelectedDestination] = [
        {
          ...(step3Data.token ? { token: step3Data.token } : {}),
          ...(step3Data.token_file ? { token_file: step3Data.token_file } : {}),
          ...(step3Data.path ? { path: step3Data.path } : {}),
          ...(step3Data.structured ? { structured: step3Data.structured } : {}),
          ...(step3Data.zip ? { zip: step3Data.zip } : {}),
          ...(step3Data.keep ? { keep: step3Data.keep } : {}),
          ...(step3Data.bare ? { bare: step3Data.bare } : {}),
          ...(step3Data.lfs ? { lfs: step3Data.lfs } : {}),
        },
      ];
    }
  
    const yamlConfig = jsYaml.dump(configData, { skipInvalid: true });
  
    const name = step1Data.name || 'backup-config'; 
    const fileName = `${name}.yml`;
  
    // Step 1: Save the configuration
    fetch('http://localhost:5000/api/saveConfiguration', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ yamlConfig , fileName }),
    })
      .then(response => response.json())
      .then(data => {
        console.log('Configuration saved:', data);
        
        // Step 2: Trigger the execution of the Go app
        fetch('http://localhost:5000/api/runGoApp', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({ fileName }), 
        })
          .then(response => response.json())
          .then(data => {
            console.log('Go app executed successfully:', data);
          })
          .catch(error => {
            console.error('Error executing Go app:', error);
          });
      })
      .catch(error => {
        console.error('Error saving configuration:', error);
      });
  
    console.log('Finish button clicked');
  };

  const cancel = () => {
    console.log('Cancel button clicked');

  };
  console.log("Step2 Data:", step2Data);

  return (
    <Box display="flex" justifyContent="center" alignItems="center" height="100vh">
      <Paper elevation={3} style={{ padding: '20px', width: '80%', maxWidth: '600px', overflowWrap: 'break-word' }}>
        <Typography variant="h4" gutterBottom align="center">
          Summary
        </Typography>

        <div style={{ marginBottom: '20px' }}>
          <Typography variant="h5">Name and Description:</Typography>
          {step1Data.name && <Typography><strong>Name:</strong> {step1Data.name}</Typography>}
          {step1Data.description && <Typography><strong>Description:</strong> {step1Data.description}</Typography>}
        </div>

        <div style={{ marginBottom: '20px' }}>
          <Typography variant="h5">Source Configuration:</Typography>
          {step2Data.selectedSource && <Typography><strong>Selected Source:</strong> {step2Data.selectedSource}</Typography>}
          {step2Data.token && <Typography><strong>Token:</strong> {step2Data.token}</Typography>}
          {/* ... (similarly format other source configuration fields) */}
        </div>

        <div style={{ marginBottom: '20px' }}>
          <Typography variant="h5">Destination Configuration:</Typography>
          {step3Data.SelectedDestination && <Typography><strong>Selected Destination:</strong> {step3Data.SelectedDestination}</Typography>}
          {step3Data.token && <Typography><strong>Token:</strong> {step3Data.token}</Typography>}
          {/* ... (similarly format other destination configuration fields) */}
        </div>

        <div style={{ marginBottom: '20px' }}>
          <Typography variant="h5">Scheduler Configuration:</Typography>
          {step4Data.selectedDays && <Typography><strong>Days:</strong> {step4Data.selectedDays}</Typography>}
          {step4Data.selectedTime && <Typography><strong>Time:</strong> {step4Data.selectedTime}</Typography>}
          {step4Data.cronExpression && <Typography><strong>Cron Expression:</strong> {step4Data.cronExpression}</Typography>}
        </div>

        <Box mt={2} display="flex" justifyContent="center">
          <Button variant="contained" color="primary" onClick={finish} style={{ marginRight: '10px' }}>
            Finish
          </Button>
          <Button variant="contained" color="secondary" onClick={cancel}>
            Cancel
          </Button>
        </Box>
      </Paper>
    </Box>
  );
};

ReactDOM.render(<Summary />, document.getElementById('root'));

export default Summary;