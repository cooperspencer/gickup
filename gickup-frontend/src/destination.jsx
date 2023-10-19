import React, { useState } from 'react';
import TextField from '@mui/material/TextField';
import Checkbox from '@mui/material/Checkbox';
import FormControlLabel from '@mui/material/FormControlLabel';
import Button from '@mui/material/Button';
import Select from '@mui/material/Select';
import MenuItem from '@mui/material/MenuItem';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import Typography from '@mui/material/Typography';

function DestinationConfig() {
  const [selectedDestination, setSelectedDestination] = useState('');
  const [destinationConfig, setDestinationConfig] = useState({
    token: '',
    token_file: '',
    user: '',
    url: '',
    createorg: false,
    lfs: false,
    visibility: '',
    organization: '',
    force: false,
    sshkey: '',
    path: '',
    structured: false,
    zip: false,
    keep: 0,
    bare: false,
  });

  const handleDestinationChange = (e) => {
    setSelectedDestination(e.target.value);
  };

  const handleInputChange = (e) => {
    const { name, value, type, checked } = e.target;
    const inputValue = type === 'checkbox' ? checked : value;
    setDestinationConfig({
      ...destinationConfig,
      [name]: inputValue,
    });
  };

  const handleSave = () => {
    const configurationObject = {
      // ... configuration data
    };
  
    fetch('http://localhost:5000/api/saveConfiguration', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(configurationObject),
    })
      .then(response => response.json())
      .then(data => {
        console.log('Configuration saved:', data);
      })
      .catch(error => {
        console.error('Error saving configuration:', error);
      });
  };

  return (
    <div style={{ padding: 20 }}>
      <Typography variant="h4" gutterBottom>
        Destination Configuration
      </Typography>
      <FormControl fullWidth variant="outlined" margin="normal">
        <InputLabel id="destination-label">Select Destination</InputLabel>
        <Select
          labelId="destination-label"
          id="destination"
          value={selectedDestination}
          label="Select Destination"
          onChange={handleDestinationChange}
        >
          <MenuItem value="">Select</MenuItem>
          <MenuItem value="Gitea">Gitea</MenuItem>
          <MenuItem value="Gogs">Gogs</MenuItem>
          <MenuItem value="Gitlab">Gitlab</MenuItem>
          <MenuItem value="Github">Github</MenuItem>
          <MenuItem value="Sourcehut">Sourcehut</MenuItem>
          <MenuItem value="Onedev">Onedev</MenuItem>
          <MenuItem value="Localpath">Localpath</MenuItem>
        </Select>
      </FormControl>

      {selectedDestination && (
        <div style={{ marginTop: 20 }}>
          <TextField
            fullWidth
            label="Token"
            variant="outlined"
            name="token"
            value={destinationConfig.token}
            onChange={handleInputChange}
            margin="normal"
          />
          <TextField
            fullWidth
            label="Token File"
            variant="outlined"
            name="token_file"
            value={destinationConfig.token_file}
            onChange={handleInputChange}
            margin="normal"
          />
          {/* Add more destination-specific fields using TextField, Checkbox, etc. */}
        </div>
      )}

      <Button variant="contained" color="primary" onClick={handleSave} style={{ marginTop: 20 }}>
        Save Configuration
      </Button>
    </div>
  );
}

export default DestinationConfig;
