import React, { useState } from 'react';
import Checkbox from '@mui/material/Checkbox';
import FormControlLabel from '@mui/material/FormControlLabel';
import TextField from '@mui/material/TextField';
import FormGroup from '@mui/material/FormGroup'; 
import Grid from '@mui/material/Grid';
import Typography from '@mui/material/Typography';

function SchedulerConfig() {
  const [selectedDays, setSelectedDays] = useState([]);
  const [selectedTime, setSelectedTime] = useState('12:00');

  const weekdays = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];

  const handleDayToggle = (day) => {
    if (selectedDays.includes(day)) {
      setSelectedDays(selectedDays.filter(selectedDay => selectedDay !== day));
    } else {
      setSelectedDays([...selectedDays, day]);
    }
  };

  const handleTimeChange = (newTime) => {
    setSelectedTime(newTime);
  };

  const getCronExpression = () => {
    const daysOfWeek = selectedDays.length > 0 ? selectedDays.join(',') : '*';
    const [hour, minute] = selectedTime.split(':');
    return `0 ${minute} ${hour} ? * ${daysOfWeek}`;
  };

  return (
    <div style={{ padding: 20 }}>
      <Typography variant="h4" gutterBottom>
        Scheduler Configuration
      </Typography>
      <Grid container spacing={2}>
        <Grid item xs={12}>
          <Typography variant="h6">Select Days of the Week:</Typography>
          <FormGroup row>
            {weekdays.map((day, index) => (
              <FormControlLabel
                key={index}
                control={
                  <Checkbox
                    checked={selectedDays.includes(day)}
                    onChange={() => handleDayToggle(day)}
                  />
                }
                label={day}
              />
            ))}
          </FormGroup>
        </Grid>
        <Grid item xs={12}>
          <Typography variant="h6">Select Time:</Typography>
          <TextField
            type="time"
            variant="outlined"
            fullWidth
            value={selectedTime}
            onChange={(e) => handleTimeChange(e.target.value)}
            InputLabelProps={{ shrink: true }}
          />
        </Grid>
        <Grid item xs={12}>
          <Typography variant="h6">Cron Expression:</Typography>
          <TextField
            variant="outlined"
            fullWidth
            value={getCronExpression()}
            InputProps={{
              readOnly: true,
            }}
          />
        </Grid>
      </Grid>
    </div>
  );
}

export default SchedulerConfig;
