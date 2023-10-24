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
    let updatedConfig = {
      ...destinationConfig,
      [name]: inputValue,
    };

    if (selectedDestination === 'Localpath') {
     
      const localpathFields = ['path', 'structured', 'zip', 'keep', 'bare', 'lfs'];
      
      localpathFields.forEach((field) => {
        if (!Object.keys(destinationConfig).includes(field)) {
          updatedConfig[field] = ''; 
        }
      });
    }

    setDestinationConfig(updatedConfig);
  };

  const handleNext = () => {
    const { token, token_file, path, structured, zip, keep, bare, lfs} = destinationConfig;
    const SelectedDestination = selectedDestination;
  
    if (
      (SelectedDestination === 'Localpath' && path.trim() === '') ||
      (SelectedDestination !== 'Localpath' && (token.trim() === '' || token_file.trim() === ''))
    ) {
      setError('Please fill out all required fields.');
    } else {
      const data = {
        token: token,
        token_file: token_file,
        SelectedDestination,
        path: path, 
        structured: structured,
        zip: zip,
        keep: keep,
        bare: bare,
        lfs: lfs,
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

      {selectedDestination !== 'Localpath' && (
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
          
        </div>
      )}

      {selectedDestination === 'Localpath' && (
        <div style={{ marginTop: 20 }}>
          <TextField
            fullWidth
            label="Path"
            variant="outlined"
            name="path"
            value={destinationConfig.path}
            onChange={handleInputChange}
            margin="normal"
            multiline
            helperText="Export this path from Docker with a volume to make it accessible and more permanent"
          />
          <FormControlLabel
            control={<Checkbox checked={destinationConfig.structured} onChange={handleInputChange} name="structured" />}
            label="Structured"
            multiline
            helperText="structures repos output like hostersite/user|organization/repo"
          />
          <FormControlLabel
            control={<Checkbox checked={destinationConfig.zip} onChange={handleInputChange} name="zip" />}
            label="Zip"
            multiline
            helperText="Zips the Backup repo"
          />
          <TextField
            fullWidth
            label="Retention"
            variant="outlined"
            name="Keep"
            value={destinationConfig.keep}
            onChange={handleInputChange}
            margin="normal"
            multiline
            helperText="only keeps x backups"
          />
          <FormControlLabel
            control={<Checkbox checked={destinationConfig.bare} onChange={handleInputChange} name="bare" />}
            label="Bare"
            multiline
            helperText="Backup the repositories as bare"
          />
          <FormControlLabel
            control={<Checkbox checked={destinationConfig.lfs} onChange={handleInputChange} name="lfs" />}
            label="LFS"
            multiline
            helperText="(LFS) replaces large files media samples with text pointers"
          />
        </div>
      )}

      <Button variant="contained" color="primary" onClick={handlePrevious} style={{ marginRight: '10px', marginTop: '1rem' }}>
        Previous
      </Button>
      <Button variant="contained" color="primary" onClick={handleNext} style={{ marginTop: '1rem' }}>
        Next
      </Button>
    </div>
  );
}

export default DestinationConfig;
