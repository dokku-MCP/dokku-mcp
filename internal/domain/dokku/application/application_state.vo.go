package application

import (
	"fmt"
	"slices"
)

// StateValue représente les valeurs possibles de l'état d'une application
type StateValue string

const (
	StateCreated     StateValue = "created"     // Application créée mais pas encore déployée
	StateDeploying   StateValue = "deploying"   // Déploiement en cours
	StateDeployed    StateValue = "deployed"    // Déployée mais pas nécessairement en cours d'exécution
	StateRunning     StateValue = "running"     // En cours d'exécution
	StateStopped     StateValue = "stopped"     // Arrêtée
	StateError       StateValue = "error"       // En erreur
	StateMaintenance StateValue = "maintenance" // En maintenance
)

// ApplicationState représente l'état d'une application avec logique de transition
type ApplicationState struct {
	value StateValue
}

// NewApplicationState crée un nouvel état d'application
func NewApplicationState(state StateValue) (*ApplicationState, error) {
	if !isValidState(state) {
		return nil, fmt.Errorf("état d'application invalide: %s", state)
	}

	return &ApplicationState{value: state}, nil
}

// MustNewApplicationState crée un état en paniquant en cas d'erreur
func MustNewApplicationState(state StateValue) *ApplicationState {
	appState, err := NewApplicationState(state)
	if err != nil {
		panic(fmt.Sprintf("impossible de créer l'état %s: %v", state, err))
	}
	return appState
}

// Value retourne la valeur de l'état
func (as *ApplicationState) Value() StateValue {
	return as.value
}

// String implémente fmt.Stringer
func (as *ApplicationState) String() string {
	return string(as.value)
}

// Equal compare deux états
func (as *ApplicationState) Equal(other *ApplicationState) bool {
	if other == nil {
		return false
	}
	return as.value == other.value
}

// IsRunning vérifie si l'état est "running"
func (as *ApplicationState) IsRunning() bool {
	return as.value == StateRunning
}

// IsDeployed vérifie si l'application est déployée (deployed, running, stopped)
func (as *ApplicationState) IsDeployed() bool {
	return as.value == StateDeployed ||
		as.value == StateRunning ||
		as.value == StateStopped
}

// IsDeploying vérifie si l'état est "deploying"
func (as *ApplicationState) IsDeploying() bool {
	return as.value == StateDeploying
}

// IsError vérifie si l'état est "error"
func (as *ApplicationState) IsError() bool {
	return as.value == StateError
}

// IsMaintenance vérifie si l'état est "maintenance"
func (as *ApplicationState) IsMaintenance() bool {
	return as.value == StateMaintenance
}

// CanTransitionTo vérifie si une transition vers un autre état est possible
func (as *ApplicationState) CanTransitionTo(newState *ApplicationState) bool {
	if newState == nil {
		return false
	}

	return isValidTransition(as.value, newState.value)
}

// GetPossibleTransitions retourne les transitions possibles depuis l'état actuel
func (as *ApplicationState) GetPossibleTransitions() []StateValue {
	transitions, exists := stateTransitions[as.value]
	if !exists {
		return []StateValue{}
	}

	// Retourner une copie pour éviter les mutations
	result := make([]StateValue, len(transitions))
	copy(result, transitions)
	return result
}

// Description retourne une description lisible de l'état
func (as *ApplicationState) Description() string {
	descriptions := map[StateValue]string{
		StateCreated:     "Application créée, prête pour le déploiement",
		StateDeploying:   "Déploiement en cours",
		StateDeployed:    "Application déployée",
		StateRunning:     "Application en cours d'exécution",
		StateStopped:     "Application arrêtée",
		StateError:       "Application en erreur",
		StateMaintenance: "Application en maintenance",
	}

	if desc, exists := descriptions[as.value]; exists {
		return desc
	}
	return string(as.value)
}

// Machine à états pour les transitions valides
var stateTransitions = map[StateValue][]StateValue{
	StateCreated: {
		StateDeploying, // Commencer un déploiement
		StateError,     // Erreur de configuration
	},
	StateDeploying: {
		StateDeployed, // Déploiement réussi
		StateRunning,  // Déploiement et démarrage réussis
		StateError,    // Échec du déploiement
	},
	StateDeployed: {
		StateRunning,     // Démarrer l'application
		StateStopped,     // Application déployée mais pas démarrée
		StateDeploying,   // Nouveau déploiement
		StateError,       // Erreur au démarrage
		StateMaintenance, // Passer en maintenance
	},
	StateRunning: {
		StateStopped,     // Arrêter l'application
		StateDeploying,   // Nouveau déploiement
		StateError,       // Erreur durant l'exécution
		StateMaintenance, // Passer en maintenance
	},
	StateStopped: {
		StateRunning,     // Redémarrer l'application
		StateDeploying,   // Nouveau déploiement
		StateError,       // Erreur au redémarrage
		StateMaintenance, // Passer en maintenance
	},
	StateError: {
		StateCreated,     // Réinitialiser l'application
		StateDeploying,   // Tentative de redéploiement
		StateRunning,     // Correction et redémarrage
		StateStopped,     // Arrêter pour investigation
		StateMaintenance, // Passer en maintenance pour correction
	},
	StateMaintenance: {
		StateDeploying, // Fin de maintenance, redéploiement
		StateRunning,   // Fin de maintenance, redémarrage
		StateStopped,   // Fin de maintenance, arrêt
		StateError,     // Problème durant la maintenance
	},
}

// isValidState vérifie si une valeur d'état est valide
func isValidState(state StateValue) bool {
	validStates := []StateValue{
		StateCreated, StateDeploying, StateDeployed,
		StateRunning, StateStopped, StateError, StateMaintenance,
	}

	return slices.Contains(validStates, state)
}

// isValidTransition vérifie si une transition d'état est valide
func isValidTransition(from, to StateValue) bool {
	transitions, exists := stateTransitions[from]
	if !exists {
		return false
	}

	return slices.Contains(transitions, to)
}
