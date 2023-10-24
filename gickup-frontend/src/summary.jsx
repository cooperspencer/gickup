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
        },}
      ];
    }

    if (step3Data.SelectedDestination) {
      configData.destination[step3Data.SelectedDestination] = [
        {
          ...(step3Data.token ? { token: step3Data.token } : {}),
          ...(step3Data.token_file ? { token_file: step3Data.token_file } : {}),
        }
      ];
    }

    const yamlConfig = jsYaml.dump(configData, { skipInvalid: true });
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
  console.log("Step2 Data:", step2Data);

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
        {step2Data.selectedSource && <Typography>Selected Source: {step2Data.selectedSource}</Typography>}
        {step2Data.token && <Typography>Token: {step2Data.token}</Typography>}
        {step2Data.user && <Typography>User: {step2Data.user}</Typography>}
        {step2Data.ssh && <Typography>SSH: {step2Data.ssh}</Typography>}
        {step2Data.sshkey && <Typography>SSH Key: {step2Data.sshkey}</Typography>}
        {step2Data.excluderepos && <Typography>Exclude Repos: {step2Data.excluderepos}</Typography>}
        {step2Data.excludeorgs && <Typography>Exclude Organizations: {step2Data.excludeorgs}</Typography>}
        {step2Data.includerepos && <Typography>Include Repos: {step2Data.includerepos}</Typography>}
        {step2Data.includeorgs && <Typography>Include Organizations: {step2Data.includeorgs}</Typography>}
        {step2Data.wiki && <Typography>Include Wiki: {step2Data.wiki}</Typography>}
        {step2Data.starred && <Typography>Include Starred Repos: {step2Data.starred}</Typography>}
        {step2Data.filterstars && <Typography>Stars Filter: {step2Data.filterstars}</Typography>}
        {step2Data.lastactivity && <Typography>Last Activity Filter: {step2Data.lastactivity}</Typography>}
        {step2Data.excludearchived && <Typography>Exclude Archived Repos</Typography>}
        {step2Data.languages && <Typography>Languages Filter: {step2Data.languages}</Typography>}
        {step2Data.excludeforks && <Typography>Exclude Forked Repos</Typography>}


        <Typography variant="h6">Destination Configuration:</Typography>
        {step3Data.SelectedDestination && <Typography>Selected Destination: {step3Data.SelectedDestination}</Typography>}
        {step3Data.token && <Typography>Token: {step3Data.token}</Typography>}
        {step3Data.token_file && <Typography>Token File: {step3Data.token_file}</Typography>}

        <Typography variant="h6">Scheduler Configuration:</Typography>
        {step4Data.selectedDays && (<Typography>Days: {step4Data.selectedDays}</Typography>)}
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
