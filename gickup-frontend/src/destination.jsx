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

function DestinationConfig(props) {
  const [selectedDestination, setSelectedDestination] = useState('');
  const [error, setError] = useState('');
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

  const handleNext = () => {
  const { token, token_file ,  } = destinationConfig;
  const SelectedDestination = selectedDestination; 
 
  if (token.trim() === '' && token_file.trim() === '') {
    setError('Either User or Token is required for selected source');
  } 
  else {
    const data = {
      token: token,
      token_file: token_file,
      SelectedDestination,
  };

  console.log('Selected destination:', SelectedDestination);
  localStorage.setItem('Step3', JSON.stringify(data));
  
    props.nextStep();
  }
};

  const handlePrevious = () => {
    
    if (props.previousStep) {
      props.previousStep();
    }
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

          <Button variant="contained" color="primary" onClick={handlePrevious} style={{  marginRight: '10px' , marginTop: '1rem' }}>
            Previous
          </Button>
          <Button variant="contained" color="primary" onClick={handleNext} style={{ marginTop: '1rem' }}>
            Next
          </Button>
    </div>
  );
}

export default DestinationConfig;
