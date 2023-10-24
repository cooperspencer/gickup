import React, { useState } from 'react';
import MenuItem from '@mui/material/MenuItem';
import TextField from '@mui/material/TextField';
import Checkbox from '@mui/material/Checkbox';
import FormControlLabel from '@mui/material/FormControlLabel';
import Button from '@mui/material/Button';
import Typography from '@mui/material/Typography';

function SourceConfig(props) {
  const [selectedSource, setSelectedSource ] = useState('');
  const [error, setError] = useState('');
  const [sourceConfig, setSourceConfig] = useState({
    token: '',
    user: '',
    username: '',
    password: '',
    ssh: false,
    sshkey: '',
    excluderepos: '',
    excludeorgs: '',
    includerepos: '',
    includeorgs: '',
    wiki: false,
    starred: false,
    filterstars: '',
    lastactivity: '',
    excludearchived: false,
    languages: '',
    excludeforks: false,
    
  });

  const handleSourceChange = (e) => {
    setSelectedSource(e.target.value);
      
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
    setSourceConfig(prevState => ({
      ...prevState,
      [name]: type === 'checkbox' ? checked : value,
    }));
  };

  const handleNext = () => {
    const { token, user, ssh, sshkey , 
      excluderepos, excludeorgs, 
      includerepos, includeorgs, 
      wiki, starred, filterstars, 
      lastactivity, excludearchived,
      languages,excludeforks } = sourceConfig;
  
    if (token.trim() === '' && user.trim() === '') {
      setError('Either User or Token is required for the selected source');
    } else if (ssh && sshkey.trim() === '') {
      setError('SSH Key Path is required for SSH authentication.');
    } else {
      const data = {
        token: token,
        user: user,
        ssh: ssh,
        sshkey: sshkey,
        selectedSource: selectedSource,
        excluderepos: excluderepos,
        excludeorgs: excludeorgs,
        includerepos: includerepos,
        includeorgs: includeorgs,
        wiki: wiki,
        starred: starred,
        filterstars: filterstars,
        lastactivity: lastactivity,
        excludearchived: excludearchived,
        languages: languages,
        excludeforks: excludeforks, 
      };
      localStorage.setItem('Step2', JSON.stringify(data));
      console.log('Selected Source:', data);
      props.nextStep();
    }
  };
  
  const handlePrevious = () => {
    props.previousStep(); 
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
          <TextField
            fullWidth
            label="Exclude Repos"
            variant="outlined"
            name="excluderepos"
            value={sourceConfig.excluderepos}
            onChange={handleInputChange}
            margin="normal"
            multiline
          />
          <TextField
            fullWidth
            label="Include Repos"
            variant="outlined"
            name="includerepos"
            value={sourceConfig.includerepos}
            onChange={handleInputChange}
            margin="normal"
            multiline
          />
          <TextField
            fullWidth
            label="Exclude Organizations"
            variant="outlined"
            name="excludeorgs"
            value={sourceConfig.excludeorgs}
            onChange={handleInputChange}
            margin="normal"
            multiline
          />
          <TextField
            fullWidth
            label="Include Organizations"
            variant="outlined"
            name="includeorgs"
            value={sourceConfig.includeorgs}
            onChange={handleInputChange}
            margin="normal"
            multiline
            
          />
          <FormControlLabel
            control={
              <Checkbox
                checked={sourceConfig.wiki}
                onChange={handleInputChange}
                name="wiki"
                color="primary"
              />
            }
            label="Include Wiki"
          />
          <FormControlLabel
            control={
              <Checkbox
                checked={sourceConfig.starred}
                onChange={handleInputChange}
                name="starred"
                color="primary"
              />
            }
            label="Include Starred Repos"
          />
          <TextField
            fullWidth
            label="Stars Filter"
            variant="outlined"
            name="filterstars"
            value={sourceConfig.stars}
            onChange={handleInputChange}
            margin="normal"
            multiline
            helperText = "only clone repos with 100 stars ie: 100"
          />
          <TextField
            fullWidth
            label="Last Activity Filter"
            variant="outlined"
            name="lastactivity"
            value={sourceConfig.lastactivity}
            onChange={handleInputChange}
            margin="normal"
            multiline
            helperText="only clone repos which had activity during the last year ie: 1yr"
          />
          <FormControlLabel
            control={
              <Checkbox
                checked={sourceConfig.excludearchived}
                onChange={handleInputChange}
                name="excludearchived"
                color="primary"
              />
            }
            label="Exclude Archived Repos"
          />
          <TextField
            fullWidth
            label="Languages Filter"
            variant="outlined"
            name="languages"
            value={sourceConfig.languages}
            onChange={handleInputChange}
            margin="normal"
            multiline
            helperText="only backup repositories with the following languages ie: -go -java"
          />
          <FormControlLabel
            control={
              <Checkbox
                checked={sourceConfig.excludeforks}
                onChange={handleInputChange}
                name="excludeforks"
                color="primary"
              />
            }
            label="Exclude Forked Repos"
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
          <Button variant="contained" color="primary" onClick={handlePrevious} style={{  marginRight: '10px' , marginTop: '1rem' }}>
            Previous
          </Button>
          <Button variant="contained" color="primary" onClick={handleNext} style={{ marginTop: '1rem' }}>
            Next
          </Button>
        </div>
      )}
    </div>
  );
}

export default SourceConfig;
