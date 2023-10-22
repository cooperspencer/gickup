import React from 'react';
import Typography from '@mui/material/Typography';
import Button from '@mui/material/Button';

function Summary(props) {
  const { name, description, selectedSource, sourceConfig, selectedDestination, destinationConfig, selectedDays, selectedTime } = props;

  return (
    <div style={{ padding: 20 }}>
      <Typography variant="h4" gutterBottom>
        Summary
      </Typography>
      <Typography variant="h6">Name and Description:</Typography>
      <Typography>Name: {name}</Typography>
      <Typography>Description: {description}</Typography>

      <Typography variant="h6">Source Configuration:</Typography>
      <Typography>Selected Source: {selectedSource}</Typography>
      {/* Render other source configuration details here */}
      
      <Typography variant="h6">Destination Configuration:</Typography>
      <Typography>Selected Destination: {selectedDestination}</Typography>
      {/* Render other destination configuration details here */}
      
      <Typography variant="h6">Scheduler Configuration:</Typography>
      <Typography>Selected Days: {selectedDays ? selectedDays.join(', ') : 'Not specified'}</Typography>
      <Typography>Selected Time: {selectedTime}</Typography>

      <Button variant="contained" color="primary" onClick={props.finish} style={{ marginTop: '1rem' }}>
        Finish
      </Button>
      <Button variant="contained" color="secondary" onClick={props.cancel} style={{ marginTop: '1rem', marginLeft: '10px' }}>
        Cancel
      </Button>
    </div>
  );
}

export default Summary;
