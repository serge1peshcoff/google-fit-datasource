import React, { ChangeEvent, PureComponent } from 'react';
import { LegacyForms } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { DataSourceOptions, SecureJsonData } from './types';

const { SecretFormField, FormField } = LegacyForms;

interface Props extends DataSourcePluginOptionsEditorProps<DataSourceOptions> {}

interface State {}

export class ConfigEditor extends PureComponent<Props, State> {
  constructor(props: Props) {
    super(props);

    const { onOptionsChange, options } = this.props;

    onOptionsChange({
      ...options,
      secureJsonData: {
        ...options.secureJsonData,
        code: new URLSearchParams(window.location.search).get('code'),
      },
    });
  }

  onClientIdChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    const jsonData = {
      ...options.jsonData,
      clientId: event.target.value,
    };
    onOptionsChange({ ...options, jsonData });
  };

  // Secure field (only sent to the backend)
  onClientSecretChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    onOptionsChange({
      ...options,
      secureJsonData: {
        clientSecret: event.target.value,
      },
    });
  };

  onResetClientSecret = () => {
    const { onOptionsChange, options } = this.props;
    onOptionsChange({
      ...options,
      secureJsonFields: {
        ...options.secureJsonFields,
        apiKey: false,
      },
      secureJsonData: {
        ...options.secureJsonData,
        apiKey: '',
      },
    });
  };

  getGoogleAuthLink = () => {
    const authUrl = 'https://accounts.google.com/o/oauth2/v2/auth';
    const redirectUrl = window.location.origin + window.location.pathname;
    const scopes = [
      'https://www.googleapis.com/auth/userinfo.email',
      'https://www.googleapis.com/auth/userinfo.profile',
      'https://www.googleapis.com/auth/fitness.activity.read',
      'https://www.googleapis.com/auth/fitness.blood_glucose.read',
      'https://www.googleapis.com/auth/fitness.blood_pressure.read',
      'https://www.googleapis.com/auth/fitness.body.read',
      'https://www.googleapis.com/auth/fitness.heart_rate.read',
      'https://www.googleapis.com/auth/fitness.body_temperature.read',
      'https://www.googleapis.com/auth/fitness.location.read',
      'https://www.googleapis.com/auth/fitness.nutrition.read',
      'https://www.googleapis.com/auth/fitness.oxygen_saturation.read',
      'https://www.googleapis.com/auth/fitness.reproductive_health.read',
      'https://www.googleapis.com/auth/fitness.sleep.read',
    ];
    const clientId = this.props.options.jsonData.clientId;

    return `${authUrl}?scope=${scopes.join(
      ' '
    )}&include_granted_scopes=true&access_type=offline&response_type=code&redirect_uri=${redirectUrl}&client_id=${clientId}`;
  };

  render() {
    const { options } = this.props;
    const { jsonData, secureJsonFields } = options;
    const secureJsonData = (options.secureJsonData || {}) as SecureJsonData;

    const googleAuthLink = this.getGoogleAuthLink();

    return (
      <div className="gf-form-group">
        <div className="gf-form">
          <FormField
            label="Client ID"
            labelWidth={6}
            inputWidth={20}
            onChange={this.onClientIdChange}
            value={jsonData.clientId || ''}
          />
        </div>

        <div className="gf-form-inline">
          <div className="gf-form">
            <SecretFormField
              isConfigured={(secureJsonFields && secureJsonFields.apiKey) as boolean}
              value={secureJsonData.clientSecret || ''}
              label="Client secret"
              labelWidth={6}
              inputWidth={20}
              onReset={this.onResetClientSecret}
              onChange={this.onClientSecretChange}
            />
          </div>
        </div>

        <div className="gf-form-group">
          <a type="button" href={googleAuthLink}>
            <img src="public/plugins/serge1peshcoff-googlefit-datasource/img/logo.png" />
          </a>
        </div>
      </div>
    );
  }
}
