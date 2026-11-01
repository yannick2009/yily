// Generates a random secret of the selected type and hands it back via onGenerate.
import { createSignal } from "solid-js";
import { Button } from "../ui";
import { t } from "../../lib/i18n";
import {
  generateApiKey,
  generateBase64,
  generateHex,
  generatePassword,
  generateUrlSafe,
  generateUuid,
} from "../../lib/generate";

type GenType = "password" | "apiKey" | "uuid" | "hex" | "base64" | "urlSafe";

// Map each type to its generator (static: no i18n, no reactive reads).
const GENERATORS: Record<GenType, () => string> = {
  password: generatePassword,
  apiKey: generateApiKey,
  uuid: generateUuid,
  hex: generateHex,
  base64: generateBase64,
  urlSafe: generateUrlSafe,
};

// ValueGenerator component allows users to select a type of value to generate (like password, API key, UUID, etc.) and generates a random value of that type when the "Generate" button is clicked. The generated value is passed back to the parent component via the onGenerate callback prop.
// It uses a select dropdown for type selection and a button to trigger the generation.
export function ValueGenerator(props: { onGenerate: (value: string) => void }) {
  const [type, setType] = createSignal<GenType>("password");

  return (
    <div class="flex items-center gap-2">
      <select
        class="input min-w-0"
        value={type()}
        onChange={(e) => setType(e.currentTarget.value as GenType)}
      >
        <option value="password">{t("gen.password")}</option>
        <option value="apiKey">{t("gen.apiKey")}</option>
        <option value="uuid">{t("gen.uuid")}</option>
        <option value="hex">{t("gen.hex")}</option>
        <option value="base64">{t("gen.base64")}</option>
        <option value="urlSafe">{t("gen.urlSafe")}</option>
      </select>
      <Button
        type="button"
        size="sm"
        variant="secondary"
        icon="refresh"
        onClick={() => props.onGenerate(GENERATORS[type()]())}
      >
        {t("gen.generate")}
      </Button>
    </div>
  );
}
