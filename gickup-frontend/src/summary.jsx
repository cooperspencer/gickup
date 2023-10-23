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
        '# name: ' + (step1Data.name || ''),
        '# description: ' + (step1Data.description || ''),
      ],
      source: {
        [step2Data.selectedSource]: [
          {
            token: step2Data.token,
            user: step2Data.user,
            ssh: step2Data.ssh,
            sshkey: step2Data.sshkey,
          }
        ],
      },
      destination: {
        [step3Data.selectedDestination]: [
          {
            token: step3Data.token,
            token_file: step3Data.token_file,
          }
        ],
      },
      cron: step4Data.cronExpression,
    };

    const yamlConfig = jsYaml.dump(configData);
    const blob = new Blob([yamlConfig], { type: 'application/yaml' });
    const url = window.URL.createObjectURL(blob);

    const a = document.createElement('a');
    a.href = url;
    a.download = `${step1Data.name}.yml`;
    document.body.appendChild(a);
    a.click();

    window.URL.revokeObjectURL(url);
    document.body.removeChild(a);

    console.log('Finish button clicked');
  };

  const cancel = () => {
    console.log('Cancel button clicked');
    
  };

  return (
    <Box p={3}>
      <Paper elevation={3} style={{ padding: '20px' }}>
        <Typography variant="h4" gutterBottom align="center">
          Summary
        </Typography>

        <Typography variant="h6">Name and Description:</Typography>
        {step1Data.name && <Typography>Name: {step1Data.name}</Typography>}
        {step1Data.description && <Typography>Description: {step1Data.description}</Typography>}

        <Typography variant="h6">Source Configuration:</Typography>
        {step2Data.Source && <Typography>Selected Source: {step2Data.selectedSource}</Typography>}
        {step2Data.token && <Typography>Token: {step2Data.token}</Typography>}
        {step2Data.user && <Typography>User: {step2Data.user}</Typography>}
        {step2Data.ssh && <Typography>SSH: {step2Data.ssh}</Typography>}
        {step2Data.sshkey && <Typography>SSH Key: {step2Data.sshkey}</Typography>}

        <Typography variant="h6">Destination Configuration:</Typography>
        {step3Data.Destination && <Typography>Selected Destination: {step3Data.selectedDestination}</Typography>}
        {step3Data.token && <Typography>Token: {step3Data.token}</Typography>}
        {step3Data.token_file && <Typography>Token File: {step3Data.token_file}</Typography>}

        <Typography variant="h6">Scheduler Configuration:</Typography>
        {step4Data.selectedDays && (<Typography>Days: {step4Data.selectedDays}</Typography>        )}
        {step4Data.selectedTime && <Typography>Time: {step4Data.selectedTime}</Typography>}
        {step4Data.cronExpression && <Typography>Cron Expression: {step4Data.cronExpression}</Typography>}
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
