import React, { useState } from 'react';
import MenuItem from '@mui/material/MenuItem';
import TextField from '@mui/material/TextField';
import Checkbox from '@mui/material/Checkbox';
import FormControlLabel from '@mui/material/FormControlLabel';
import Button from '@mui/material/Button';
import Typography from '@mui/material/Typography';


function SourceConfig() {
  const [selectedSource, setSelectedSource] = useState('');
  const [sourceConfig, setSourceConfig] = useState({
    token: '',
    user: '',
    username: '',
    password: '',
    ssh: false,
    sshkey: '',
  });

  const handleSourceChange = (e) => {
    setSelectedSource(e.target.value);
    // Reset sourceConfig when source changes
    setSourceConfig({
      token: '',
      user: '',
      username: '',
      password: '',
      ssh: false,
      sshkey: '',
    });
  };

  const handleInputChange = (e) => {
    const { name, value, type, checked } = e.target;
    const inputValue = type === 'checkbox' ? checked : value;
    setSourceConfig({
      ...sourceConfig,
      [name]: inputValue,
    });
  };

  const handleSave = () => {
    // Handle saving sourceConfig based on the selected source option
    // For example, send it to your backend API to save it to the file.
    console.log('Configuration saved:', sourceConfig);
  };

  return (
    <div>
      <Typography variant="h4" gutterBottom>
        Source Configuration
      </Typography>
      <TextField
        select
        fullWidth
        label="Source"
        variant="outlined"
        value={selectedSource}
        onChange={handleSourceChange}
        margin="normal"
      >
        <MenuItem value="">Select a source</MenuItem>
        <MenuItem value="Github">GitHub</MenuItem>
        <MenuItem value="Gitea">Gitea</MenuItem>
        <MenuItem value="Gogs">Gogs</MenuItem>
        <MenuItem value="Gitlab">Gitlab</MenuItem>
        <MenuItem value="Bitbucket">Bitbucket</MenuItem>
        <MenuItem value="Onedev">Onedev</MenuItem>
        <MenuItem value="Sourcehut">Sourcehut</MenuItem>
        <MenuItem value="Other">Other</MenuItem>
      </TextField>

      {selectedSource && (
        <div>
      <Typography variant="h4" gutterBottom>
        Source
      </Typography>
          <TextField
            fullWidth
            label="Token"
            variant="outlined"
            name="token"
            value={sourceConfig.token}
            onChange={handleInputChange}
            margin="normal"
          />
          <TextField
            fullWidth
            label="User"
            variant="outlined"
            name="user"
            value={sourceConfig.user}
            onChange={handleInputChange}
            margin="normal"
          />
          <TextField
            fullWidth
            label="Username"
            variant="outlined"
            name="username"
            value={sourceConfig.username}
            onChange={handleInputChange}
            margin="normal"
          />
          <TextField
            fullWidth
            label="Password"
            variant="outlined"
            type="password"
            name="password"
            value={sourceConfig.password}
            onChange={handleInputChange}
            margin="normal"
          />
          <FormControlLabel
            control={
              <Checkbox
                checked={sourceConfig.ssh}
                onChange={handleInputChange}
                name="ssh"
                color="primary"
              />
            }
            label="SSH"
          />
          {sourceConfig.ssh && (
            <TextField
              fullWidth
              label="SSH Key Path"
              variant="outlined"
              name="sshkey"
              value={sourceConfig.sshkey}
              onChange={handleInputChange}
              margin="normal"
            />
          )}

          <Button variant="contained" color="primary" onClick={handleSave} style={{ marginTop: '1rem' }}>
            Save Configuration
          </Button>
        </div>
      )}
    </div>
  );
}

export default SourceConfig;
