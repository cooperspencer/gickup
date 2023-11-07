import React, { useState } from 'react';
import StepZilla from 'react-stepzilla';
import Step1 from './Step1'; 
import Step2 from './Step2';
import Step3 from './Step3'; 
import Step4 from './Step4'; 
import Step5 from './Step5'; 

const steps = [
  { name: 'Name and Description', component: <Step1 /> },
  { name: 'Source Configuration', component: <Step2 /> },
  { name: 'Destination Configuration', component: <Step3 /> },
  { name: 'Scheduler Configuration', component: <Step4 /> },
  { name: 'Summary and Finish', component: <Step5 /> },
];

const BackupWizard = () => {
  const [currentStep, setCurrentStep] = useState(0);

  const handleNext = (data) => {
    // Handle data from current step if needed
    console.log('Data from current step:', data);
    
    // Proceed to the next step
    setCurrentStep(currentStep + 1);
  };

  return (
    <div className="wizard-container">
      <StepZilla 
        steps={steps}
        currentStep={currentStep}
        onNext={handleNext}
        showSteps={false}  
        showNavigation={false}  
        stepsNavigation={false}  
      />
    </div>
  );
};

export default BackupWizard;
