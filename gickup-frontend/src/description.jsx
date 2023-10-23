import React, { useState } from 'react';
import { Box, TextField, Typography, Button } from '@mui/material';

const DescriptionStep = (props) => {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [error, setError] = useState('');

  const handleNameChange = (event) => {
    setName(event.target.value);
  };

  const handleDescriptionChange = (event) => {
    setDescription(event.target.value);
  };

  const handleNextClick = () => {
    if (name.trim() === '' || description.trim() === '') {
      setError('Both Name and Description fields cannot be empty.');
    } else {
      setError('');
      const data = {
        name: name,
        description: description,
      };
      localStorage.setItem('Step1', JSON.stringify(data));
      props.nextStep();
    }
  };

  return (
    <Box sx={{ maxWidth: 400, margin: 'auto' }}>
      <Typography variant="h5" gutterBottom>
        Step 1: Name and Description
      </Typography>
      <TextField
        fullWidth
        margin="normal"
        label="Name"
        variant="outlined"
        value={name}
        onChange={handleNameChange}
        error={error !== ''}
        helperText={error}
      />
      <TextField
        fullWidth
        margin="normal"
        label="Description"
        variant="outlined"
        multiline
        rows={4}
        value={description}
        onChange={handleDescriptionChange}
      />
      <Button variant="contained" color="primary" onClick={handleNextClick} style={{ marginTop: '1rem' }}>
        Next
      </Button>
    </Box>
  );
};

export default DescriptionStep;
